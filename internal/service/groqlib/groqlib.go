package groqlib

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/arkoes07/llm/internal/config"
	"github.com/conneroisu/groq-go"
	"github.com/conneroisu/groq-go/pkg/groqerr"
	"github.com/conneroisu/groq-go/pkg/schema"
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

type callOption func(*groq.ChatCompletionRequest) error

func withTools(ts ...tools.Tool) callOption {
	return func(req *groq.ChatCompletionRequest) error {
		req.Tools = ts
		return nil
	}
}

func withJSONSchema(name string, v any) callOption {
	return func(req *groq.ChatCompletionRequest) error {
		sch, err := reflectRootSchema(v)
		if err != nil {
			return err
		}

		req.ResponseFormat = &groq.ChatResponseFormat{
			Type: groq.FormatJSONSchema,
			JSONSchema: &groq.JSONSchema{
				Name:   name,
				Schema: *sch,
				Strict: true,
			},
		}
		return nil
	}
}

func reflectRootSchema(v any) (*schema.Schema, error) {
	sch, err := schema.ReflectSchema(v)
	if err != nil {
		return nil, fmt.Errorf("reflect schema: %w", err)
	}

	if sch.Ref == "" {
		return sch, nil
	}

	name := strings.TrimPrefix(sch.Ref, "#/$defs/")
	root, ok := sch.Definitions[name]
	if !ok {
		return nil, fmt.Errorf("reflect schema: root $ref %q has no definition", sch.Ref)
	}

	delete(sch.Definitions, name)
	root.Definitions = sch.Definitions

	return root, nil
}

func (s *Service) chatCompletionsAPI(ctx context.Context, req groq.ChatCompletionRequest) (groq.ChatCompletionMessage, groq.Usage, error) {
	res, err := s.cli.ChatCompletion(ctx, req)
	if err != nil {
		return groq.ChatCompletionMessage{}, groq.Usage{}, err
	}

	if len(res.Choices) == 0 {
		return groq.ChatCompletionMessage{}, groq.Usage{}, errors.New("no choices returned")
	}

	return res.Choices[0].Message, res.Usage, nil
}

func (s *Service) chatCompletionsAPIWithRetry(ctx context.Context, messages []groq.ChatCompletionMessage, opts ...callOption) (groq.ChatCompletionMessage, groq.Usage, error) {
	const maxAttempt = 5
	backOff := time.Second

	req := groq.ChatCompletionRequest{
		Model:      groq.ChatModel(s.cfg.GroqModelName),
		Messages:   messages,
		RetryDelay: time.Second,
	}
	for _, opt := range opts {
		if err := opt(&req); err != nil {
			return groq.ChatCompletionMessage{}, groq.Usage{}, err
		}
	}

	var err error
	for attempt := 1; ; attempt++ {
		var (
			msg groq.ChatCompletionMessage
			u   groq.Usage
		)
		msg, u, err = s.chatCompletionsAPI(ctx, req)
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
