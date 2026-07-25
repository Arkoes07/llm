package groqlib

import (
	"context"
	"errors"
	"log"

	"github.com/arkoes07/llm/internal/mocktool"
	"github.com/conneroisu/groq-go"
	"github.com/conneroisu/groq-go/pkg/tools"
	"github.com/google/uuid"
)

func (s *Service) ChatWithoutSession(ctx context.Context, content string) (string, error) {
	res, err := s.chatCompletionsAPI(ctx, []groq.ChatCompletionMessage{
		{Role: "system", Content: "You are a terse assistant."},
		{Role: "user", Content: content},
	})
	if err != nil {
		return "", err
	}

	return res.Content, nil
}

func (s *Service) Chat(ctx context.Context, sessID uuid.UUID, content string) (uuid.UUID, string, error) {
	sessID, msgs := s.loadMessagesBySessionID(sessID, "You are a terse assistant.", content)

	res, err := s.chatCompletionsAPI(ctx, msgs)
	if err != nil {
		return sessID, "", err
	}

	msgs = append(msgs, res)
	s.setMessagesBySessionID(sessID, msgs)

	return sessID, res.Content, nil
}

func (s *Service) AgentChat(ctx context.Context, name string, sessID uuid.UUID, content string) (uuid.UUID, string, error) {
	if name != "weather" && name != "log_triage" {
		return sessID, "", errors.New("unknown agent")
	}

	sessID, msgs := s.loadMessagesBySessionID(sessID, getAgentSystemContent(name), content)
	for i := 0; ; i++ {
		msg, err := s.chatCompletionsAPIWithRetry(ctx, msgs, getAgentTools(name)...)
		if err != nil {
			return sessID, "", err
		}
		msgs = append(msgs, msg)

		if len(msg.ToolCalls) == 0 {
			break
		}

		for _, toolCall := range msg.ToolCalls {
			log.Printf("on iteration %d, call tool %s with args: %s\n", i+1, toolCall.Function.Name, toolCall.Function.Arguments)

			toolMsg := groq.ChatCompletionMessage{
				Role:       groq.RoleTool,
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

	s.setMessagesBySessionID(sessID, msgs)
	res := msgs[len(msgs)-1]

	return sessID, res.Content, nil
}

func getAgentSystemContent(name string) string {
	switch name {
	case "weather":
		return "You are a weather assistant. Respond to the user question and use tools if needed to answer the query."
	case "log_triage":
		return "You are a log triage assistant. Respond to the user question and use tools if needed to answer the query. Do not stop at the first plausible explanation. Before concluding, rule out alternatives: check whether the service's own resources are healthy and whether traffic is abnormal. If the evidence points to a downstream dependency, investigate that service before answering. Cite the specific evidence for and against each hypothesis. Only cite the runbook if you called search_runbook. If you did not consult a source, say the recommendation is based on general knowledge, not internal documentation."
	}
	return ""
}

func getAgentTools(name string) []tools.Tool {
	switch name {
	case "weather":
		return []tools.Tool{
			getWeatherTool,
		}
	case "log_triage":
		return []tools.Tool{
			queryLogsTool,
			getMetricsTool,
			searchRunbookTool,
		}
	}
	return nil
}
