package groqrawapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/arkoes07/llm/internal/config"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/google/uuid"
)

type Service struct {
	cfg         *config.Config
	msgsCache   map[uuid.UUID]*msgsCacheData
	msgsCacheMU *sync.Mutex
}

func New(cfg *config.Config) *Service {
	s := &Service{
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

	return s
}

type message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	Name       string     `json:"name,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []toolCall `json:"tool_calls,omitempty"`
}

type tool struct {
	Type     string   `json:"type"`
	Function toolFunc `json:"function"`
}

type toolCall struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"`
	Function toolFunc `json:"function"`
}

type toolFunc struct {
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Arguments   string    `json:"arguments,omitempty"`
	Parameters  funcParam `json:"parameters,omitzero"`
}

type funcParam struct {
	Type       string              `json:"type"`
	Properties map[string]property `json:"properties"`
	Required   []string            `json:"required"`
}

type property struct {
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	Enum        []any  `json:"enum,omitempty"`
}

type choice struct {
	Message message `json:"message"`
}

type chatCompletionsAPIReq struct {
	Model          string          `json:"model"`
	Messages       []message       `json:"messages"`
	Tools          []tool          `json:"tools,omitempty"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
}

type responseFormat struct {
	Type       string          `json:"type"`
	JSONSchema *jsonSchemaSpec `json:"json_schema,omitempty"`
}

type jsonSchemaSpec struct {
	Name   string             `json:"name"`
	Schema *jsonschema.Schema `json:"schema"`
	Strict bool               `json:"strict"`
}

type chatCompletionsAPIResp struct {
	ID      string   `json:"id"`
	Model   string   `json:"model"`
	Choices []choice `json:"choices"`
	Usage   usage    `json:"usage"`
}

type usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

type apiError struct {
	StatusCode int
	Body       string
	RetryAfter time.Duration
}

func (e *apiError) Error() string {
	return fmt.Sprintf("groq %d: %s", e.StatusCode, e.Body)
}

type callOption func(*chatCompletionsAPIReq) error

func withTools(ts ...tool) callOption {
	return func(req *chatCompletionsAPIReq) error {
		req.Tools = ts
		return nil
	}
}

func withJSONSchema(name string, v any) callOption {
	return func(req *chatCompletionsAPIReq) error {
		t := reflect.TypeOf(v)
		if t == nil {
			return fmt.Errorf("infer %q schema: nil value", name)
		}

		for t.Kind() == reflect.Pointer {
			t = t.Elem()
		}

		sch, err := jsonschema.ForType(t, nil)
		if err != nil {
			return fmt.Errorf("infer %q schema: %w", name, err)
		}

		req.ResponseFormat = &responseFormat{
			Type: "json_schema",
			JSONSchema: &jsonSchemaSpec{
				Name:   name,
				Schema: sch,
				Strict: true,
			},
		}
		return nil
	}
}

func (s *Service) chatCompletionsAPI(ctx context.Context, chatCompletionsAPIReq chatCompletionsAPIReq) (message, usage, error) {
	body, err := json.Marshal(chatCompletionsAPIReq)
	if err != nil {
		return message{}, usage{}, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://api.groq.com/openai/v1/chat/completions",
		bytes.NewReader(body),
	)
	if err != nil {
		return message{}, usage{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.cfg.GroqAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return message{}, usage{}, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return message{}, usage{}, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return message{}, usage{}, &apiError{
			StatusCode: resp.StatusCode,
			Body:       string(raw),
			RetryAfter: parseRetryAfter(resp.Header.Get("retry-after")),
		}
	}

	var res chatCompletionsAPIResp
	err = json.Unmarshal(raw, &res)
	if err != nil {
		return message{}, usage{}, err
	}

	if len(res.Choices) == 0 {
		return message{}, usage{}, errors.New("no choices returned")
	}

	return res.Choices[0].Message, res.Usage, nil
}

func (s *Service) chatCompletionsAPIWithRetry(ctx context.Context, messages []message, opts ...callOption) (message, usage, error) {
	const maxAttempt = 5
	backOff := time.Second

	req := chatCompletionsAPIReq{
		Model:    s.cfg.GroqModelName,
		Messages: messages,
	}
	for _, opt := range opts {
		if err := opt(&req); err != nil {
			return message{}, usage{}, err
		}
	}

	var err error
	for attempt := 1; ; attempt++ {
		var (
			msg message
			u   usage
		)
		msg, u, err = s.chatCompletionsAPI(ctx, req)
		if err == nil {
			return msg, u, nil
		}

		var ae *apiError
		if errors.As(err, &ae) {
			if ae.StatusCode != http.StatusTooManyRequests && ae.StatusCode < 500 {
				return message{}, usage{}, err
			}
		}

		if attempt == maxAttempt {
			break
		}

		wait := backOff
		if ae != nil && ae.RetryAfter > 0 {
			wait = ae.RetryAfter
		}
		log.Printf("attempt %d failed (%v), retrying in %v\n", attempt, err, wait)

		t := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			t.Stop()
			return message{}, usage{}, ctx.Err()
		case <-t.C:
		}

		backOff *= 2
	}

	return message{}, usage{}, fmt.Errorf("after %d attempts: %w", maxAttempt, err)
}

func parseRetryAfter(h string) time.Duration {
	if h == "" {
		return 0
	}
	if secs, err := strconv.ParseFloat(h, 64); err == nil {
		return time.Duration(secs * float64(time.Second))
	}
	return 0
}
