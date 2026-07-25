package service

import (
	"context"

	"github.com/google/uuid"
)

type Service interface {
	ChatWithoutSession(ctx context.Context, content string) (string, error)
	Chat(ctx context.Context, sessID uuid.UUID, content string) (uuid.UUID, string, error)
	AgentChat(ctx context.Context, name string, sessID uuid.UUID, content string) (uuid.UUID, string, error)
}
