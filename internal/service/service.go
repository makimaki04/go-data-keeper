package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/makimaki04/go-data-keeper.git/internal/repository"
	"go.uber.org/zap"
)

type Authorization interface {
	RegisterUser(ctx context.Context, login string, password string) (id uuid.UUID, accessToken JWTToken, err error)
	LoginUser(ctx context.Context, login string, password string) (accessToken JWTToken, err error)
	GenerateToken(id uuid.UUID) (accessToken JWTToken, err error)
}

type Service struct {
	Authorization
}

func NewService(repo *repository.Repository, secret string, logger *zap.SugaredLogger) *Service {
	return &Service{
		Authorization: NewAuthService(repo.Authorization, secret, logger),
	}
}