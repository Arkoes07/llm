package grogapi

import (
	"time"

	"github.com/google/uuid"
)

type msgsCacheData struct {
	msgs          []message
	createdAtUnix int64
}

func (s *Service) getMessagesBySessionID(sessID uuid.UUID) []message {
	s.msgsMU.Lock()
	defer s.msgsMU.Unlock()

	data, ok := s.msgsCache[sessID]
	if !ok {
		return nil
	}

	return data.msgs
}

func (s *Service) appendMessagesBySessionID(sessID uuid.UUID, msgs ...message) {
	s.msgsMU.Lock()
	defer s.msgsMU.Unlock()

	if _, ok := s.msgsCache[sessID]; !ok {
		s.msgsCache[sessID] = &msgsCacheData{
			createdAtUnix: time.Now().Unix(),
		}
	}
	s.msgsCache[sessID].msgs = append(s.msgsCache[sessID].msgs, msgs...)
}

func (s *Service) deleteInvalidMessagesCache() {
	s.msgsMU.Lock()
	defer s.msgsMU.Unlock()

	nowUnix := time.Now().Unix()
	for sessID, data := range s.msgsCache {
		if nowUnix-data.createdAtUnix > int64(15*time.Minute) {
			delete(s.msgsCache, sessID)
		}
	}
}
