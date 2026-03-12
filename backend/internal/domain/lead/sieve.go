package lead

import "context"

type Sieve interface {
	DetectLead(ctx context.Context, text string, senderKey string) (isLead bool, score float32, err error)
	AddReferencePoint(ctx context.Context, text string, isLead bool) error
}
