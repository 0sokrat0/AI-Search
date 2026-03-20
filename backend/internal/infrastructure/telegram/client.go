package telegram

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"MRG/internal/config"

	"github.com/gotd/contrib/middleware/ratelimit"
	"github.com/gotd/td/bin"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/telegram/auth/qrlogin"
	"github.com/gotd/td/tg"
	"github.com/gotd/td/tgerr"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

type Client struct {
	acc         AccountConfig
	log         *zap.Logger
	tgClient    *telegram.Client
	dispatcher  *tg.UpdateDispatcher
	msgChan     chan PendingMsg
	handler     MessageHandler
	workerCount int
	workerBatch int
	workerFlush time.Duration

	onPasswordNeeded func()
	onAuthorized     func(AccountIdentity)
	onActivated      func(AccountIdentity)

	authCodeChan  chan string
	authPassChan  chan string
	qrChan        chan string
	authErrorChan chan error
}

type AccountIdentity struct {
	Phone    string
	Name     string
	Username string
	UserID   int64
}

func NewClient(cfg *config.Config, acc AccountConfig, log *zap.Logger, db *mongo.Database, handler MessageHandler) (*Client, error) {
	dispatcher := tg.NewUpdateDispatcher()

	c := &Client{
		acc:           acc,
		log:           log,
		dispatcher:    &dispatcher,
		msgChan:       make(chan PendingMsg, 512),
		handler:       handler,
		workerCount:   4,
		workerBatch:   16,
		workerFlush:   2 * time.Second,
		authCodeChan:  make(chan string, 1),
		authPassChan:  make(chan string, 1),
		qrChan:        make(chan string, 1),
		authErrorChan: make(chan error, 1),
	}
	RegisterRuntimeConfig(cap(c.msgChan), c.workerCount, c.workerBatch, c.workerFlush)

	app := pickRandomApp()
	s := NewMongoSessionStorage(db, acc.Phone)
	mtprotoLog := log.Named("mtproto").WithOptions(zap.IncreaseLevel(zap.WarnLevel))

	opts := telegram.Options{
		Logger:         mtprotoLog,
		SessionStorage: s,
		UpdateHandler:  &dispatcher,
		Middlewares: []telegram.Middleware{
			ratelimit.New(rate.Every(5*time.Second), 2),
			floodWaitPanic(log, acc.Phone),
		},
		Device: app.Device,
	}

	if acc.Proxy != "" || acc.ProxyFallback != "" {
		resolver, err := buildResolver(acc.Proxy, acc.ProxyFallback)
		if err != nil {
			return nil, fmt.Errorf("proxy setup: %w", err)
		}
		if resolver != nil {
			opts.Resolver = resolver
		}
	}

	c.tgClient = telegram.NewClient(app.AppID, app.AppHash, opts)

	dispatcher.OnNewMessage(func(ctx context.Context, e tg.Entities, u *tg.UpdateNewMessage) error {
		msg, ok := u.Message.(*tg.Message)
		if !ok || msg.Message == "" || msg.Out {
			return nil
		}
		c.enqueueMessage(e, msg)
		return nil
	})

	dispatcher.OnNewChannelMessage(func(ctx context.Context, e tg.Entities, u *tg.UpdateNewChannelMessage) error {
		msg, ok := u.Message.(*tg.Message)
		if !ok || msg.Message == "" || msg.Out {
			return nil
		}
		c.enqueueMessage(e, msg)
		return nil
	})

	return c, nil
}

func (c *Client) Run(ctx context.Context) error {
	c.log.Debug("starting telegram client runtime",
		zap.String("phone", c.acc.Phone),
		zap.Int("worker_count", c.workerCount),
		zap.Int("worker_batch", c.workerBatch),
		zap.Duration("worker_flush", c.workerFlush),
	)
	go c.startWorkers(ctx)

	return c.tgClient.Run(ctx, func(ctx context.Context) error {
		status, err := c.tgClient.Auth().Status(ctx)
		if err != nil {
			return err
		}
		if !status.Authorized {
			return fmt.Errorf("account not authorized")
		}
		if status.User == nil {
			return fmt.Errorf("authorized account info is missing")
		}

		identity := AccountIdentity{
			Phone:    status.User.Phone,
			Name:     strings.TrimSpace(status.User.FirstName + " " + status.User.LastName),
			Username: status.User.Username,
			UserID:   status.User.ID,
		}

		if c.acc.Phone == "" || strings.HasPrefix(c.acc.Phone, "pending_qr") {
			c.acc.Phone = identity.Phone
			c.log.Info("discovered phone after auth", zap.String("phone", identity.Phone))
		}

		if c.onAuthorized != nil {
			c.onAuthorized(identity)
		}
		if c.onActivated != nil {
			c.onActivated(identity)
		}

		c.log.Info("account listening for updates", zap.String("phone", c.acc.Phone), zap.Int64("user_id", identity.UserID))
		<-ctx.Done()
		return nil
	})
}

func (c *Client) Authenticate(ctx context.Context, useQR bool) error {
	return c.tgClient.Run(ctx, func(ctx context.Context) error {
		if useQR {
			return c.authQR(ctx)
		}
		return c.authPhone(ctx)
	})
}

func (c *Client) authQR(ctx context.Context) error {
	c.log.Info("starting QR authentication", zap.String("phone", c.acc.Phone))
	loggedIn := qrlogin.OnLoginToken(c.dispatcher)
	_, err := c.tgClient.QR().Auth(ctx, loggedIn, func(ctx context.Context, token qrlogin.Token) error {
		c.log.Info("QR code generated", zap.String("url", token.URL()), zap.String("phone", c.acc.Phone))
		select {
		case c.qrChan <- token.URL():
		default:
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, auth.ErrPasswordAuthNeeded) || strings.Contains(err.Error(), "SESSION_PASSWORD_NEEDED") {
			c.log.Info("2FA password required for QR login", zap.String("phone", c.acc.Phone))
			if c.onPasswordNeeded != nil {
				c.onPasswordNeeded()
			}
			select {
			case pass := <-c.authPassChan:
				c.log.Info("2FA password received, authenticating", zap.String("phone", c.acc.Phone))
				if _, err = c.tgClient.Auth().Password(ctx, pass); err != nil {
					c.log.Error("2FA password authentication failed", zap.Error(err), zap.String("phone", c.acc.Phone))
					return err
				}
				c.log.Info("2FA password authentication successful", zap.String("phone", c.acc.Phone))
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		c.log.Error("QR authentication failed", zap.Error(err), zap.String("phone", c.acc.Phone))
		return err
	}
	c.log.Info("QR authentication successful", zap.String("phone", c.acc.Phone))
	return nil
}

func (c *Client) authPhone(ctx context.Context) error {
	c.log.Info("starting phone authentication", zap.String("phone", c.acc.Phone))
	flow := auth.NewFlow(c, auth.SendCodeOptions{})
	if err := flow.Run(ctx, c.tgClient.Auth()); err != nil {
		c.log.Error("phone authentication failed", zap.Error(err), zap.String("phone", c.acc.Phone))
		return err
	}
	c.log.Info("phone authentication successful", zap.String("phone", c.acc.Phone))
	return nil
}

func (c *Client) Phone(ctx context.Context) (string, error) {
	return c.acc.Phone, nil
}

func (c *Client) Password(ctx context.Context) (string, error) {
	c.log.Info("2FA password requested by flow", zap.String("phone", c.acc.Phone))
	if c.onPasswordNeeded != nil {
		c.onPasswordNeeded()
	}
	select {
	case pass := <-c.authPassChan:
		c.log.Info("2FA password received for flow", zap.String("phone", c.acc.Phone))
		return pass, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (c *Client) AcceptTermsOfService(ctx context.Context, tos tg.HelpTermsOfService) error {
	c.log.Warn("telegram sign-up terms of service requested", zap.String("phone", c.acc.Phone))
	return &auth.SignUpRequired{TermsOfService: tos}
}

func (c *Client) SignUp(ctx context.Context) (auth.UserInfo, error) {
	c.log.Warn("telegram sign-up requested for unknown account", zap.String("phone", c.acc.Phone))
	return auth.UserInfo{}, errors.New("telegram sign-up is not supported")
}

func (c *Client) Code(ctx context.Context, sentCode *tg.AuthSentCode) (string, error) {
	c.log.Info("SMS code requested by flow", zap.String("phone", c.acc.Phone), zap.Any("type", sentCode.Type))
	select {
	case code := <-c.authCodeChan:
		c.log.Info("SMS code received for flow", zap.String("phone", c.acc.Phone))
		return code, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (c *Client) ProvideCode(code string) {
	select {
	case c.authCodeChan <- code:
		return
	default:
	}
	select {
	case <-c.authCodeChan:
	default:
	}
	select {
	case c.authCodeChan <- code:
	default:
	}
}

func (c *Client) ProvidePassword(pass string) {
	select {
	case c.authPassChan <- pass:
		return
	default:
	}
	select {
	case <-c.authPassChan:
	default:
	}
	select {
	case c.authPassChan <- pass:
	default:
	}
}

func (c *Client) GetQRChan() <-chan string {
	return c.qrChan
}

func (c *Client) flushBatch(ctx context.Context, batch []PendingMsg) {
	c.log.Debug("flushing telegram message batch",
		zap.String("phone", c.acc.Phone),
		zap.Int("batch_size", len(batch)),
	)
	if err := c.handler.Handle(ctx, batch); err != nil {
		MarkHandlerError(err)
		c.log.Error("telegram parser batch failed",
			zap.Error(err),
			zap.String("phone", c.acc.Phone),
			zap.Int("batch_size", len(batch)),
		)
		return
	}
	MarkBatchProcessed(len(batch))
	c.log.Debug("telegram message batch processed",
		zap.String("phone", c.acc.Phone),
		zap.Int("batch_size", len(batch)),
	)
}

func (c *Client) enqueueMessage(e tg.Entities, msg *tg.Message) {
	chatTitle := ""
	switch peer := msg.PeerID.(type) {
	case *tg.PeerChannel:
		if ch, found := e.Channels[peer.ChannelID]; found && ch != nil {
			chatTitle = strings.TrimSpace(ch.Title)
		}
	case *tg.PeerChat:
		if ch, found := e.Chats[peer.ChatID]; found && ch != nil {
			chatTitle = strings.TrimSpace(ch.Title)
		}
	case *tg.PeerUser:
		peerMeta := c.resolveSenderMeta(e, peer)
		if peerMeta.Name != "" {
			chatTitle = peerMeta.Name
		} else if peerMeta.Username != "" {
			chatTitle = peerMeta.Username
		}
	}

	senderID := c.getPeerID(msg.FromID)
	meta := c.resolveSenderMeta(e, msg.FromID)

	if senderID == 0 {
		senderID = c.getPeerID(msg.PeerID)
		if senderID != 0 {
			meta = c.resolveSenderMeta(e, msg.PeerID)
		}
	}

	senderName := meta.Name
	if senderName == "" {
		senderName = meta.Username
	}
	if senderName == "" && chatTitle != "" {
		senderName = chatTitle
	}
	if senderName == "" && senderID != 0 {
		senderName = fmt.Sprintf("id:%d", senderID)
	}

	_, isDM := msg.PeerID.(*tg.PeerUser)

	pending := PendingMsg{
		MessageID:      int64(msg.ID),
		SenderID:       senderID,
		SenderName:     senderName,
		SenderUsername: meta.Username,
		SenderPeerType: c.getPeerType(msg.FromID, msg.PeerID),
		IsScam:         meta.IsScam,
		IsFake:         meta.IsFake,
		IsPremium:      meta.IsPremium,
		Text:           msg.Message,
		ChatTitle:      chatTitle,
		PeerID:         c.getPeerID(msg.PeerID),
		Date:           msg.Date,
		IsDM:           isDM,
	}

	select {
	case c.msgChan <- pending:
		MarkEnqueued()
		c.log.Debug("telegram message enqueued",
			zap.String("phone", c.acc.Phone),
			zap.Int64("peer_id", pending.PeerID),
			zap.Int64("sender_id", pending.SenderID),
			zap.Int64("message_id", pending.MessageID),
			zap.Bool("is_dm", pending.IsDM),
		)
	default:
		MarkDropped()
		c.log.Warn("message queue overflow: dropping telegram message",
			zap.String("phone", c.acc.Phone),
			zap.Int64("peer_id", pending.PeerID),
			zap.Int64("message_id", pending.MessageID),
		)
	}
}

func (c *Client) startWorkers(ctx context.Context) {
	var wg sync.WaitGroup
	for range c.workerCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.runWorker(ctx)
		}()
	}
	<-ctx.Done()
	c.log.Debug("stopping telegram workers", zap.String("phone", c.acc.Phone))
	wg.Wait()
}

func (c *Client) runWorker(ctx context.Context) {
	batch := make([]PendingMsg, 0, c.workerBatch)
	ticker := time.NewTicker(c.workerFlush)
	defer ticker.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}
		c.flushBatch(ctx, batch)
		batch = batch[:0]
	}

	for {
		select {
		case <-ctx.Done():
			c.log.Debug("telegram worker context canceled", zap.String("phone", c.acc.Phone))
			flush()
			return
		case msg := <-c.msgChan:
			batch = append(batch, msg)
			if len(batch) >= c.workerBatch {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

func floodWaitPanic(log *zap.Logger, phone string) telegram.MiddlewareFunc {
	return func(next tg.Invoker) telegram.InvokeFunc {
		return func(ctx context.Context, input bin.Encoder, output bin.Decoder) error {
			err := next.Invoke(ctx, input, output)
			if d, ok := tgerr.AsFloodWait(err); ok {
				log.Error("FLOOD_WAIT received", zap.Duration("wait", d), zap.String("phone", phone))
				panic(fmt.Sprintf("FLOOD_WAIT %s for %s", d, phone))
			}
			return err
		}
	}
}

func (c *Client) getPeerID(peer tg.PeerClass) int64 {
	switch p := peer.(type) {
	case *tg.PeerUser:
		return p.UserID
	case *tg.PeerChat:
		return p.ChatID
	case *tg.PeerChannel:
		return p.ChannelID
	default:
		return 0
	}
}

func (c *Client) getPeerType(primary tg.PeerClass, fallback tg.PeerClass) string {
	for _, peer := range []tg.PeerClass{primary, fallback} {
		switch peer.(type) {
		case *tg.PeerUser:
			return "user"
		case *tg.PeerChat:
			return "chat"
		case *tg.PeerChannel:
			return "channel"
		}
	}
	return ""
}

type senderMeta struct {
	Name      string
	Username  string
	IsScam    bool
	IsFake    bool
	IsPremium bool
}

func (c *Client) resolveSenderMeta(e tg.Entities, peer tg.PeerClass) senderMeta {
	var m senderMeta
	switch p := peer.(type) {
	case *tg.PeerUser:
		if u, ok := e.Users[p.UserID]; ok && u != nil {
			m.Name = strings.TrimSpace(u.FirstName + " " + u.LastName)
			m.Username = strings.TrimSpace(u.Username)
			m.IsScam = u.Scam
			m.IsFake = u.Fake
			m.IsPremium = u.Premium
		}
	case *tg.PeerChat:
		if ch, ok := e.Chats[p.ChatID]; ok && ch != nil {
			m.Name = strings.TrimSpace(ch.Title)
		}
	case *tg.PeerChannel:
		if ch, ok := e.Channels[p.ChannelID]; ok && ch != nil {
			m.Name = strings.TrimSpace(ch.Title)
			m.Username = strings.TrimSpace(ch.Username)
		}
	}
	return m
}
