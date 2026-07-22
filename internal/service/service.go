package service

import (
	"github.com/google/uuid"
)

type Service interface {
	ChatWithoutSession(content string) (string, error)
	Chat(sessID uuid.UUID, content string) (uuid.UUID, string, error)
	AgentChat(sessID uuid.UUID, content string) (uuid.UUID, string, error)
}
