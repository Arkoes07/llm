package service

import (
	"context"

	"github.com/arkoes07/llm/internal/domain"
	"github.com/google/uuid"
)

type Service interface {
	ChatWithoutSession(ctx context.Context, content string) (string, error)
	Chat(ctx context.Context, sessID uuid.UUID, content string) (uuid.UUID, string, error)
	AgentChat(ctx context.Context, name domain.AgentName, sessID uuid.UUID, content string) (uuid.UUID, string, error)
	GenerateStudyPlan(ctx context.Context, param domain.StudyPlanParam) (domain.StudyPlan, error)
}
