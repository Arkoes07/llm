package groqlib

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/arkoes07/llm/internal/config"
	"github.com/conneroisu/groq-go"
	"github.com/conneroisu/groq-go/pkg/tools"
	"github.com/google/uuid"
)

type Service struct {
	cli       *groq.Client
	cfg       *config.Config
	msgsCache map[uuid.UUID]*msgsCacheData
	msgsMU    *sync.Mutex
}

func New(cfg *config.Config) (*Service, error) {
	cli, err := groq.NewClient(cfg.GroqAPIKey)
	if err != nil {
		return nil, err
	}

	s := &Service{
		cli:       cli,
		cfg:       cfg,
		msgsCache: make(map[uuid.UUID]*msgsCacheData),
		msgsMU:    &sync.Mutex{},
	}

	go func() {
		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			s.deleteExpiredSessions()
		}
	}()

	return s, nil
}

func (s *Service) chatCompletionsAPI(messages []groq.ChatCompletionMessage, tools ...tools.Tool) (groq.ChatCompletionMessage, error) {
	res, err := s.cli.ChatCompletion(context.Background(), groq.ChatCompletionRequest{
		Model:    groq.ChatModel(s.cfg.GrogModelName),
		Messages: messages,
		Tools:    tools,
	})
	if err != nil {
		return groq.ChatCompletionMessage{}, err
	}

	if len(res.Choices) == 0 {
		return groq.ChatCompletionMessage{}, errors.New("no choices returned")
	}

	return res.Choices[0].Message, nil
}

func (s *Service) chatCompletionsAPIWithRetry(messages []groq.ChatCompletionMessage, tools ...tools.Tool) (groq.ChatCompletionMessage, error) {
	const maxAttempt = 5
	backOff := time.Second

	var err error
	for attempt := 0; ; attempt++ {
		var msg groq.ChatCompletionMessage
		msg, err = s.chatCompletionsAPI(messages, tools...)
		if err == nil {
			return msg, nil
		}

		if attempt == maxAttempt {
			break
		}

		wait := backOff
		log.Printf("attempt %d failed (%v), retrying in %v", attempt+1, err, wait)

		time.Sleep(wait)
		backOff *= 2
	}

	return groq.ChatCompletionMessage{}, fmt.Errorf("after %d attempts: %w", maxAttempt, err)
}
