package settings

import (
	"context"
	"strconv"
	"time"

	"MRG/internal/domain/message"
	"MRG/internal/infrastructure/storage/settings"
)

type UseCase struct {
	store    *settings.Store
	messages message.Repository
}

func New(store *settings.Store, messages message.Repository) *UseCase {
	return &UseCase{store: store, messages: messages}
}

func (uc *UseCase) GetAll(ctx context.Context) (map[string]string, error) {
	return uc.store.GetAll(ctx)
}

func (uc *UseCase) Update(ctx context.Context, patch map[string]string) error {
	floatKeys := []string{
		"lead_threshold",
		"trader_threshold",
		"merchant_threshold",
		"ps_offer_threshold",
	}
	for _, k := range floatKeys {
		if v, ok := patch[k]; ok {
			f, err := strconv.ParseFloat(v, 64)
			if err != nil || f < 0 || f > 1 {
				return ErrInvalidThreshold
			}
		}
	}
	if v, ok := patch["sender_window_seconds"]; ok {
		n, err := strconv.Atoi(v)
		if err != nil || n < 5 || n > 3600 {
			return ErrInvalidWindow
		}
	}
	for _, k := range []string{"noise_cleanup_enabled", "show_multi_account_badges"} {
		if v, ok := patch[k]; ok {
			if _, err := strconv.ParseBool(v); err != nil {
				return ErrInvalidThreshold
			}
		}
	}
	return uc.store.SetAll(ctx, patch)
}

func (uc *UseCase) CleanupNoise(ctx context.Context, tenantID string, olderThan time.Duration) (int64, error) {
	if uc.messages == nil {
		return 0, nil
	}
	return uc.messages.DeleteOldNoise(ctx, tenantID, olderThan)
}
