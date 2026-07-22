package grogapi

import (
	"github.com/google/uuid"

	"github.com/arkoes07/llm/internal/domain"
)

func (s *Service) ChatWithoutSession(content string) (string, error) {
	res, err := s.chat([]domain.Message{
		{Role: "system", Content: "You are a terse assistant."},
		{Role: "user", Content: content},
	})
	if err != nil {
		return "", err
	}

	return res.Content, nil
}

func (s *Service) Chat(sessID uuid.UUID, content string) (uuid.UUID, string, error) {
	sessID, exiMsgs, newMsgs := s.loadMessagesBySessionID(sessID, "You are a terse assistant.", content)

	res, err := s.chat(append(exiMsgs, newMsgs...))
	if err != nil {
		return sessID, "", err
	}

	newMsgs = append(newMsgs, res)
	s.appendMessagesBySessionID(sessID, newMsgs...)

	return sessID, res.Content, nil
}

func (s *Service) AgentChat(sessID uuid.UUID, content string) (uuid.UUID, string, error) {
	sessID, exiMsgs, newMsgs := s.loadMessagesBySessionID(sessID, "You are a weather assistant. Respond to the user question and use tools if needed to answer the query.", content)
	msgs := append(exiMsgs, newMsgs...)

	for {
		msg, err := s.chat(msgs, getWeatherTool)
		if err != nil {
			return sessID, "", err
		}

		msgs = append(msgs, msg)
		newMsgs = append(newMsgs, msg)

		if len(msg.ToolCalls) == 0 {
			break
		}

		for _, toolCall := range msg.ToolCalls {
			if toolCall.Function.Name != "get_weather" {
				continue
			}

			toolMsg := domain.Message{
				Role:       "tool",
				Content:    getWeather(toolCall.Function.Arguments),
				ToolCallID: toolCall.ID,
				Name:       toolCall.Function.Name,
			}

			msgs = append(msgs, toolMsg)
			newMsgs = append(newMsgs, toolMsg)
		}
	}

	s.appendMessagesBySessionID(sessID, newMsgs...)
	res := newMsgs[len(newMsgs)-1]

	return sessID, res.Content, nil
}

func (s *Service) loadMessagesBySessionID(sessID uuid.UUID, systemContent, userContent string) (uuid.UUID, []domain.Message, []domain.Message) {
	if sessID == uuid.Nil {
		sessID = uuid.New()
	}

	exiMsgs := s.getMessagesBySessionID(sessID)
	newMsgs := make([]domain.Message, 0)

	if len(exiMsgs) == 0 {
		newMsgs = append(newMsgs, domain.Message{
			Role:    "system",
			Content: systemContent,
		})
	}

	newMsgs = append(newMsgs, domain.Message{
		Role:    "user",
		Content: userContent,
	})

	return sessID, exiMsgs, newMsgs
}
