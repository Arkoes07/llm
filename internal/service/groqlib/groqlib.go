package groqlib

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/arkoes07/llm/internal/config"
	"github.com/conneroisu/groq-go"
	"github.com/conneroisu/groq-go/pkg/groqerr"
	"github.com/conneroisu/groq-go/pkg/tools"
	"github.com/google/uuid"
)

type Service struct {
	cli         *groq.Client
	cfg         *config.Config
	msgsCache   map[uuid.UUID]*msgsCacheData
	msgsCacheMU *sync.Mutex
}

func New(cfg *config.Config) (*Service, error) {
	cli, err := groq.NewClient(cfg.GroqAPIKey)
	if err != nil {
		return nil, err
	}

	s := &Service{
		cli:         cli,
		cfg:         cfg,
		msgsCache:   make(map[uuid.UUID]*msgsCacheData),
		msgsCacheMU: &sync.Mutex{},
	}

	go func() {
		ticker := time.NewTicker(sessionTTL)
		defer ticker.Stop()

		for range ticker.C {
			s.deleteExpiredSessions()
		}
	}()

	return s, nil
}

func (s *Service) chatCompletionsAPI(ctx context.Context, messages []groq.ChatCompletionMessage, tools ...tools.Tool) (groq.ChatCompletionMessage, groq.Usage, error) {
	res, err := s.cli.ChatCompletion(ctx, groq.ChatCompletionRequest{
		Model:      groq.ChatModel(s.cfg.GroqModelName),
		Messages:   messages,
		Tools:      tools,
		RetryDelay: time.Second,
	})
	if err != nil {
		return groq.ChatCompletionMessage{}, groq.Usage{}, err
	}

	if len(res.Choices) == 0 {
		return groq.ChatCompletionMessage{}, groq.Usage{}, errors.New("no choices returned")
	}

	return res.Choices[0].Message, res.Usage, nil
}

func (s *Service) chatCompletionsAPIWithRetry(ctx context.Context, messages []groq.ChatCompletionMessage, tools ...tools.Tool) (groq.ChatCompletionMessage, groq.Usage, error) {
	const maxAttempt = 5
	backOff := time.Second

	var err error
	for attempt := 1; ; attempt++ {
		var (
			msg groq.ChatCompletionMessage
			u   groq.Usage
		)
		msg, u, err = s.chatCompletionsAPI(ctx, messages, tools...)
		if err == nil {
			return msg, u, nil
		}

		if status := httpStatus(err); status != 0 &&
			status != http.StatusTooManyRequests && status < 500 {
			return groq.ChatCompletionMessage{}, groq.Usage{}, err
		}

		if attempt == maxAttempt {
			break
		}

		wait := backOff
		log.Printf("attempt %d failed (%v), retrying in %v\n", attempt, err, wait)

		t := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			t.Stop()
			return groq.ChatCompletionMessage{}, groq.Usage{}, ctx.Err()
		case <-t.C:
		}

		backOff *= 2
	}

	return groq.ChatCompletionMessage{}, groq.Usage{}, fmt.Errorf("after %d attempts: %w", maxAttempt, err)
}

func httpStatus(err error) int {
	var apiErr *groqerr.APIError
	if errors.As(err, &apiErr) {
		return apiErr.HTTPStatusCode
	}

	var reqErr *groqerr.ErrRequest
	if errors.As(err, &reqErr) {
		return reqErr.HTTPStatusCode
	}

	return 0
}
