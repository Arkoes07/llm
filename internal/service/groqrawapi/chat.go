package groqrawapi

import (
	"context"
	"fmt"
	"log"

	"github.com/arkoes07/llm/internal/domain"
	"github.com/arkoes07/llm/internal/mocktool"
	"github.com/google/uuid"
)

func logUsage(sessID uuid.UUID, iteration int, u usage) {
	log.Printf("usage: session=%s iteration=%d prompt_tokens=%d completion_tokens=%d\n",
		sessID, iteration, u.PromptTokens, u.CompletionTokens)
}

func (s *Service) ChatWithoutSession(ctx context.Context, content string) (string, error) {
	res, u, err := s.chatCompletionsAPIWithRetry(ctx, []message{
		{Role: "system", Content: domain.ChatSystemPrompt},
		{Role: "user", Content: content},
	})
	if err != nil {
		return "", err
	}
	logUsage(uuid.Nil, 1, u)

	return res.Content, nil
}

func (s *Service) Chat(ctx context.Context, sessID uuid.UUID, content string) (uuid.UUID, string, error) {
	sessID, sess, release := s.acquireSession(sessID)
	defer release()

	msgs := sess.load(domain.ChatSystemPrompt, content)

	res, u, err := s.chatCompletionsAPIWithRetry(ctx, msgs)
	if err != nil {
		return sessID, "", err
	}
	logUsage(sessID, 1, u)

	sess.store(append(msgs, res))

	return sessID, res.Content, nil
}

func (s *Service) AgentChat(ctx context.Context, name domain.AgentName, sessID uuid.UUID, content string) (uuid.UUID, string, error) {
	if !name.Valid() {
		return sessID, "", fmt.Errorf("unknown agent %q", name)
	}

	sessID, sess, release := s.acquireSession(sessID)
	defer release()

	msgs := sess.load(name.SystemPrompt(), content)
	for i := 0; ; i++ {
		msg, u, err := s.chatCompletionsAPIWithRetry(ctx, msgs, withTools(getAgentTools(name)...))
		if err != nil {
			return sessID, "", err
		}
		logUsage(sessID, i+1, u)
		msgs = append(msgs, msg)

		if len(msg.ToolCalls) == 0 {
			break
		}

		for _, toolCall := range msg.ToolCalls {
			log.Printf("on iteration %d, call tool %s with args: %s\n", i+1, toolCall.Function.Name, toolCall.Function.Arguments)

			toolMsg := message{
				Role:       "tool",
				ToolCallID: toolCall.ID,
				Name:       toolCall.Function.Name,
			}

			toolMsg.Content, err = mocktool.Run(toolCall.Function.Name, toolCall.Function.Arguments)
			if err != nil {
				toolMsg.Content = err.Error()
			}

			msgs = append(msgs, toolMsg)
		}
	}

	sess.store(msgs)
	res := msgs[len(msgs)-1]

	return sessID, res.Content, nil
}

func getAgentTools(name domain.AgentName) []tool {
	switch name {
	case domain.AgentWeather:
		return []tool{
			getWeatherTool,
		}
	case domain.AgentLogTriage:
		return []tool{
			queryLogsTool,
			getMetricsTool,
			searchRunbookTool,
		}
	}
	return nil
}
