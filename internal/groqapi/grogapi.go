package groqapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/arkoes07/llm/internal/config"
	"github.com/arkoes07/llm/internal/domain"
)

type Service struct {
	cfg *config.Config
}

func New(cfg *config.Config) *Service {
	return &Service{
		cfg: cfg,
	}
}

type chatReq struct {
	Model    string           `json:"model"`
	Messages []domain.Message `json:"messages"`
}

type chatResp struct {
	ID      string   `json:"id"`
	Model   string   `json:"model"`
	Choices []choice `json:"choices"`
}

type choice struct {
	Message domain.Message `json:"message"`
}

func (s *Service) Chat(messages []domain.Message) (domain.Message, error) {
	chatReq := chatReq{
		Model:    s.cfg.GrogModelName,
		Messages: messages,
	}

	body, _ := json.Marshal(chatReq)
	req, _ := http.NewRequest(
		"POST",
		"https://api.groq.com/openai/v1/chat/completions",
		bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.cfg.GroqAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return domain.Message{}, err
	}
	defer resp.Body.Close()

	var res chatResp
	raw, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(raw, &res)
	if err != nil {
		return domain.Message{}, err
	}

	if len(res.Choices) == 0 {
		return domain.Message{}, errors.New("no choices returned")
	}

	return res.Choices[0].Message, nil
}
