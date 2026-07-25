package groqlib

import (
	"slices"
	"sync"
	"time"

	"github.com/conneroisu/groq-go"
	"github.com/google/uuid"
)

const sessionTTL = 15 * time.Minute

type msgsCacheData struct {
	mu            sync.Mutex // held for a whole turn (load -> API call -> store
	msgs          []groq.ChatCompletionMessage
	createdAtUnix int64
}

func (s *Service) acquireSession(sessID uuid.UUID) (uuid.UUID, *msgsCacheData, func()) {
	if sessID == uuid.Nil {
		sessID = uuid.New()
	}

	s.msgsCacheMU.Lock()
	data, ok := s.msgsCache[sessID]
	if !ok {
		data = &msgsCacheData{createdAtUnix: time.Now().Unix()}
		s.msgsCache[sessID] = data
	}
	s.msgsCacheMU.Unlock()

	data.mu.Lock()
	return sessID, data, data.mu.Unlock
}

func (d *msgsCacheData) load(systemContent, userContent string) []groq.ChatCompletionMessage {
	msgs := slices.Clone(d.msgs)
	if len(msgs) == 0 {
		msgs = append(msgs, groq.ChatCompletionMessage{
			Role:    groq.RoleSystem,
			Content: systemContent,
		})
	}

	return append(msgs, groq.ChatCompletionMessage{
		Role:    groq.RoleUser,
		Content: userContent,
	})
}

func (d *msgsCacheData) store(msgs []groq.ChatCompletionMessage) {
	d.msgs = msgs
}

func (s *Service) deleteExpiredSessions() {
	s.msgsCacheMU.Lock()
	defer s.msgsCacheMU.Unlock()

	nowUnix := time.Now().Unix()
	for sessID, data := range s.msgsCache {
		if nowUnix-data.createdAtUnix > int64(sessionTTL.Seconds()) {
			delete(s.msgsCache, sessID)
		}
	}
}
