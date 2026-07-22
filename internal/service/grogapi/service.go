package grogapi

import (
	"sync"
	"time"

	"github.com/arkoes07/llm/internal/config"
	"github.com/arkoes07/llm/internal/domain"
	"github.com/google/uuid"
)

type Service struct {
	cfg       *config.Config
	msgsCache map[uuid.UUID]*msgsCacheData
	msgsMU    *sync.Mutex
}

func New(cfg *config.Config) *Service {
	s := &Service{
		cfg:       cfg,
		msgsCache: make(map[uuid.UUID]*msgsCacheData),
		msgsMU:    &sync.Mutex{},
	}

	go func() {
		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			s.deleteInvalidMessagesCache()
		}
	}()

	return s
}

type msgsCacheData struct {
	msgs          []domain.Message
	createdAtUnix int64
}

func (s *Service) getMessagesBySessionID(sessID uuid.UUID) []domain.Message {
	s.msgsMU.Lock()
	defer s.msgsMU.Unlock()

	data, ok := s.msgsCache[sessID]
	if !ok {
		return nil
	}

	return data.msgs
}

func (s *Service) appendMessagesBySessionID(sessID uuid.UUID, msgs ...domain.Message) {
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
