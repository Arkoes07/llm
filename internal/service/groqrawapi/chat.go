package groqrawapi

import (
	"context"
	"errors"
	"log"

	"github.com/arkoes07/llm/internal/mocktool"
	"github.com/google/uuid"
)

func logUsage(sessID uuid.UUID, iteration int, u usage) {
	log.Printf("usage: session=%s iteration=%d prompt_tokens=%d completion_tokens=%d\n",
		sessID, iteration, u.PromptTokens, u.CompletionTokens)
}

func (s *Service) ChatWithoutSession(ctx context.Context, content string) (string, error) {
	res, u, err := s.chatCompletionsAPIWithRetry(ctx, []message{
		{Role: "system", Content: "You are a terse assistant."},
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

	msgs := sess.load("You are a terse assistant.", content)

	res, u, err := s.chatCompletionsAPIWithRetry(ctx, msgs)
	if err != nil {
		return sessID, "", err
	}
	logUsage(sessID, 1, u)

	sess.store(append(msgs, res))

	return sessID, res.Content, nil
}

func (s *Service) AgentChat(ctx context.Context, name string, sessID uuid.UUID, content string) (uuid.UUID, string, error) {
	if name != "weather" && name != "log_triage" {
		return sessID, "", errors.New("unknown agent")
	}

	sessID, sess, release := s.acquireSession(sessID)
	defer release()

	msgs := sess.load(getAgentSystemContent(name), content)
	for i := 0; ; i++ {
		msg, u, err := s.chatCompletionsAPIWithRetry(ctx, msgs, getAgentTools(name)...)
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

func getAgentSystemContent(name string) string {
	switch name {
	case "weather":
		return "You are a weather assistant. Respond to the user question and use tools if needed to answer the query."
	case "log_triage":
		return "You are a log triage assistant. Respond to the user question and use tools if needed to answer the query. Do not stop at the first plausible explanation. Before concluding, rule out alternatives: check whether the service's own resources are healthy and whether traffic is abnormal. If the evidence points to a downstream dependency, investigate that service before answering. Cite the specific evidence for and against each hypothesis. Only cite the runbook if you called search_runbook. If you did not consult a source, say the recommendation is based on general knowledge, not internal documentation."
	}
	return ""
}

func getAgentTools(name string) []tool {
	switch name {
	case "weather":
		return []tool{
			getWeatherTool,
		}
	case "log_triage":
		return []tool{
			queryLogsTool,
			getMetricsTool,
			searchRunbookTool,
		}
	}
	return nil
}
