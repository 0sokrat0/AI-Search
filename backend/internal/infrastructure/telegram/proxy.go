package telegram

import (
	"context"
	"encoding/hex"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/gotd/td/telegram/dcs"
	"github.com/gotd/td/transport"
	"golang.org/x/net/proxy"
)

func buildResolver(primaryURL, fallbackURL string) (dcs.Resolver, error) {
	var resolvers []dcs.Resolver

	if primaryURL != "" {
		r, err := parseProxyURL(primaryURL)
		if err != nil {
			return nil, fmt.Errorf("primary proxy: %w", err)
		}
		resolvers = append(resolvers, r)
	}

	if fallbackURL != "" {
		r, err := parseProxyURL(fallbackURL)
		if err != nil {
			return nil, fmt.Errorf("fallback proxy: %w", err)
		}
		resolvers = append(resolvers, r)
	}

	switch len(resolvers) {
	case 0:
		return nil, nil
	case 1:
		return resolvers[0], nil
	default:
		return fallbackResolver{resolvers: resolvers}, nil
	}
}

func parseProxyURL(raw string) (dcs.Resolver, error) {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse proxy url: %w", err)
	}

	switch u.Scheme {
	case "socks5", "socks5h":
		return buildSOCKS5Resolver(u)
	case "mtproto", "mtproxy":
		secretHex := u.Query().Get("secret")
		if secretHex == "" {
			secretHex = strings.TrimPrefix(u.Path, "/")
		}
		return buildMTProxyResolver(u.Host, secretHex)
	case "https":
		if strings.HasSuffix(u.Host, "t.me") {
			q := u.Query()
			addr := net.JoinHostPort(q.Get("server"), q.Get("port"))
			return buildMTProxyResolver(addr, q.Get("secret"))
		}
		return nil, fmt.Errorf("unsupported https proxy url (only t.me/proxy links are supported)")
	default:
		return nil, fmt.Errorf("unsupported proxy scheme %q — use socks5:// or mtproto://", u.Scheme)
	}
}

func buildSOCKS5Resolver(u *url.URL) (dcs.Resolver, error) {
	if u.Host == "" {
		return nil, fmt.Errorf("socks5 host is empty")
	}

	var auth *proxy.Auth
	if u.User != nil {
		auth = &proxy.Auth{User: u.User.Username()}
		if password, ok := u.User.Password(); ok {
			auth.Password = password
		}
	}

	dialer, err := proxy.SOCKS5("tcp", u.Host, auth, proxy.Direct)
	if err != nil {
		return nil, fmt.Errorf("create socks5 dialer: %w", err)
	}

	return dcs.Plain(dcs.PlainOptions{
		Dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.Dial(network, addr)
		},
	}), nil
}

func buildMTProxyResolver(addr, secretHex string) (dcs.Resolver, error) {
	if addr == "" || addr == ":" {
		return nil, fmt.Errorf("mtproxy address is empty")
	}
	if secretHex == "" {
		return nil, fmt.Errorf("mtproxy secret is required (pass ?secret=HEXVALUE)")
	}

	secret, err := hex.DecodeString(secretHex)
	if err != nil {
		return nil, fmt.Errorf("decode mtproxy secret: %w", err)
	}

	return dcs.MTProxy(addr, secret, dcs.MTProxyOptions{})
}

type fallbackResolver struct {
	resolvers []dcs.Resolver
}

func (f fallbackResolver) Primary(ctx context.Context, dc int, list dcs.List) (transport.Conn, error) {
	return f.tryAll(func(r dcs.Resolver) (transport.Conn, error) {
		return r.Primary(ctx, dc, list)
	})
}

func (f fallbackResolver) MediaOnly(ctx context.Context, dc int, list dcs.List) (transport.Conn, error) {
	return f.tryAll(func(r dcs.Resolver) (transport.Conn, error) {
		return r.MediaOnly(ctx, dc, list)
	})
}

func (f fallbackResolver) CDN(ctx context.Context, dc int, list dcs.List) (transport.Conn, error) {
	return f.tryAll(func(r dcs.Resolver) (transport.Conn, error) {
		return r.CDN(ctx, dc, list)
	})
}

func (f fallbackResolver) tryAll(fn func(dcs.Resolver) (transport.Conn, error)) (transport.Conn, error) {
	var lastErr error
	for _, r := range f.resolvers {
		conn, err := fn(r)
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	return nil, lastErr
}