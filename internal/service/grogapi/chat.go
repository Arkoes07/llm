package grogapi

import (
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
	sessID, exiMsgs, newMsgs := s.loadMessagesBySessionID(sessID, "You are a terse assistant.", content)

	res, err := s.chatCompletionsAPI(append(exiMsgs, newMsgs...))
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
		msg, err := s.chatCompletionsAPI(msgs, getWeatherTool)
		if err != nil {
			return sessID, "", err
		}

		msgs = append(msgs, msg)
		newMsgs = append(newMsgs, msg)

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
			newMsgs = append(newMsgs, toolMsg)
		}
	}

	s.appendMessagesBySessionID(sessID, newMsgs...)
	res := newMsgs[len(newMsgs)-1]

	return sessID, res.Content, nil
}

func (s *Service) loadMessagesBySessionID(sessID uuid.UUID, systemContent, userContent string) (uuid.UUID, []message, []message) {
	if sessID == uuid.Nil {
		sessID = uuid.New()
	}

	exiMsgs := s.getMessagesBySessionID(sessID)
	newMsgs := make([]message, 0)

	if len(exiMsgs) == 0 {
		newMsgs = append(newMsgs, message{
			Role:    "system",
			Content: systemContent,
		})
	}

	newMsgs = append(newMsgs, message{
		Role:    "user",
		Content: userContent,
	})

	return sessID, exiMsgs, newMsgs
}
