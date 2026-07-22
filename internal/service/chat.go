package service

import (
	"errors"

	"github.com/arkoes07/llm/internal/domain"
	"github.com/google/uuid"
)

func (s *Service) CreateChat(content string) (string, error) {
	if content == "" {
		return "", errors.New("empty message")
	}

	res, err := s.groq.CreateChat([]domain.Message{
		{Role: "system", Content: "You are a terse assistant."},
		{Role: "user", Content: content},
	})
	if err != nil {
		return "", err
	}

	return res.Content, nil
}

func (s *Service) CreateChatWithSession(sessID uuid.UUID, content string) (uuid.UUID, string, error) {
	if content == "" {
		return sessID, "", errors.New("empty message")
	}

	if sessID == uuid.Nil {
		sessID = uuid.New()
	}

	exiMsgs := s.getMessagesBySessionID(sessID)
	newMsgs := make([]domain.Message, 0)

	if len(exiMsgs) == 0 {
		newMsgs = append(newMsgs, domain.Message{Role: "system", Content: "You are a terse assistant."})
	}

	msg := domain.Message{
		Role:    "user",
		Content: content,
	}
	newMsgs = append(newMsgs, msg)

	res, err := s.groq.CreateChat(append(exiMsgs, newMsgs...))
	if err != nil {
		return sessID, "", err
	}

	newMsgs = append(newMsgs, res)
	s.appendMessagesBySessionID(sessID, newMsgs...)

	return sessID, res.Content, nil
}
