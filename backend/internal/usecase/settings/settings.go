package settings

import (
	"context"
	"strconv"

	"MRG/internal/infrastructure/storage/settings"
)

type UseCase struct {
	store *settings.Store
}

func New(store *settings.Store) *UseCase {
	return &UseCase{store: store}
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
	return uc.store.SetAll(ctx, patch)
}
