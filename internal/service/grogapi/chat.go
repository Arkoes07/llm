package grogapi

import (
	"errors"

	"github.com/arkoes07/llm/internal/mocktool"
	"github.com/google/uuid"
)

func (s *Service) ChatWithoutSession(content string) (string, error) {
	res, err := s.chatCompletionsAPI([]message{
		{Role: "system", Content: "You are a terse assistant."},
		{Role: "user", Content: content},
	})
	if err != nil {
		return "", err
	}

	return res.Content, nil
}

func (s *Service) Chat(sessID uuid.UUID, content string) (uuid.UUID, string, error) {
	sessID, msgs := s.loadMessagesBySessionID(sessID, "You are a terse assistant.", content)

	res, err := s.chatCompletionsAPI(msgs)
	if err != nil {
		return sessID, "", err
	}

	msgs = append(msgs, res)
	s.setMessagesBySessionID(sessID, msgs)

	return sessID, res.Content, nil
}

func (s *Service) AgentChat(name string, sessID uuid.UUID, content string) (uuid.UUID, string, error) {
	if name != "weather" {
		return sessID, "", errors.New("unknown agent")
	}

	sessID, msgs := s.loadMessagesBySessionID(sessID, getAgentSystemContent(name), content)

	for {
		msg, err := s.chatCompletionsAPI(msgs, getAgentTools(name)...)
		if err != nil {
			return sessID, "", err
		}
		msgs = append(msgs, msg)

		if len(msg.ToolCalls) == 0 {
			break
		}

		for _, toolCall := range msg.ToolCalls {
			toolMsg := message{
				Role:       "tool",
				Content:    mocktool.Run(toolCall.Function.Name, toolCall.Function.Arguments),
				ToolCallID: toolCall.ID,
				Name:       toolCall.Function.Name,
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
	}
	return ""
}

func getAgentTools(name string) []tool {
	switch name {
	case "weather":
		return []tool{getWeatherTool}
	}
	return nil
}
