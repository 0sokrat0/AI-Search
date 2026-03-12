package qdrant

import (
	"MRG/internal/config"

	"github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
)

func Connect(cfg *config.Config) (*qdrant.Client, error) {
	return qdrant.NewClient(&qdrant.Config{
		Host:        cfg.Qdrant.Host,
		Port:        cfg.Qdrant.Port,
		GrpcOptions: []grpc.DialOption{},
	})
}
