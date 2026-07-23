package grogapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/arkoes07/llm/internal/config"
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
	Description string    `json:"description"`
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
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
	Tools    []tool    `json:"tools"`
}

type chatCompletionsAPIResp struct {
	ID      string   `json:"id"`
	Model   string   `json:"model"`
	Choices []choice `json:"choices"`
}

func (s *Service) chatCompletionsAPI(messages []message, tools ...tool) (message, error) {
	chatCompletionsAPIReq := chatCompletionsAPIReq{
		Model:    s.cfg.GrogModelName,
		Messages: messages,
		Tools:    tools,
	}

	body, _ := json.Marshal(chatCompletionsAPIReq)
	req, _ := http.NewRequest(
		"POST",
		"https://api.groq.com/openai/v1/chat/completions",
		bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.cfg.GroqAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return message{}, err
	}
	defer resp.Body.Close()

	var res chatCompletionsAPIResp
	raw, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(raw, &res)
	if err != nil {
		return message{}, err
	}

	if len(res.Choices) == 0 {
		return message{}, errors.New("no choices returned")
	}

	return res.Choices[0].Message, nil
}
