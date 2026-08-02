package groqlib

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/arkoes07/llm/internal/domain"
	"github.com/conneroisu/groq-go"
	"github.com/google/uuid"
)

func (s *Service) GenerateStudyPlan(ctx context.Context, param domain.StudyPlanParam) (domain.StudyPlan, error) {
	content, err := json.Marshal(param)
	if err != nil {
		return domain.StudyPlan{}, err
	}

	messages := []groq.ChatCompletionMessage{
		{Role: groq.RoleSystem, Content: domain.StudyPlanSystemPrompt},
		{Role: groq.RoleUser, Content: string(content)},
	}

	res, u, err := s.chatCompletionsAPIWithRetry(ctx, messages, withJSONSchema("study_plan", domain.StudyPlan{}))
	if err != nil {
		return domain.StudyPlan{}, err
	}
	logUsage(uuid.Nil, 1, u)

	var plan domain.StudyPlan
	if err := json.Unmarshal([]byte(res.Content), &plan); err != nil {
		return domain.StudyPlan{}, fmt.Errorf("unmarshal study plan: %w", err)
	}

	return plan, nil
}
