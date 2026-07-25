package groqlib

import (
	"time"

	"github.com/conneroisu/groq-go"
	"github.com/google/uuid"
)

type msgsCacheData struct {
	msgs          []groq.ChatCompletionMessage
	createdAtUnix int64
}

func (s *Service) getMessagesBySessionID(sessID uuid.UUID) []groq.ChatCompletionMessage {
	s.msgsMU.Lock()
	defer s.msgsMU.Unlock()

	data, ok := s.msgsCache[sessID]
	if !ok {
		return nil
	}

	return data.msgs
}

func (s *Service) setMessagesBySessionID(sessID uuid.UUID, msgs []groq.ChatCompletionMessage) {
	s.msgsMU.Lock()
	defer s.msgsMU.Unlock()

	if _, ok := s.msgsCache[sessID]; !ok {
		s.msgsCache[sessID] = &msgsCacheData{
			createdAtUnix: time.Now().Unix(),
		}
	}
	s.msgsCache[sessID].msgs = msgs
}

func (s *Service) deleteExpiredSessions() {
	s.msgsMU.Lock()
	defer s.msgsMU.Unlock()

	nowUnix := time.Now().Unix()
	for sessID, data := range s.msgsCache {
		if nowUnix-data.createdAtUnix > int64((15 * time.Minute).Seconds()) {
			delete(s.msgsCache, sessID)
		}
	}
}

func (s *Service) loadMessagesBySessionID(sessID uuid.UUID, systemContent, userContent string) (uuid.UUID, []groq.ChatCompletionMessage) {
	if sessID == uuid.Nil {
		sessID = uuid.New()
	}

	msgs := s.getMessagesBySessionID(sessID)
	if len(msgs) == 0 {
		msgs = append(msgs, groq.ChatCompletionMessage{
			Role:    groq.RoleSystem,
			Content: systemContent,
		})
	}

	msgs = append(msgs, groq.ChatCompletionMessage{
		Role:    groq.RoleUser,
		Content: userContent,
	})

	return sessID, msgs
}
