package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/arkoes07/llm/internal/httpx"
	"github.com/arkoes07/llm/internal/service"
)

const requestTimeout = 30 * time.Second

type Handler struct {
	svc *service.Service
}

func New(svc *service.Service) *Handler {
	return &Handler{
		svc: svc,
	}
}
func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Timeout(requestTimeout))

	r.Get("/healthz", h.health)
	r.Post("/chat", h.chat)
	r.Post("/chat/v2", h.chatWithSession)

	return r
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type chatWithSessionBody struct {
	SessionID *uuid.UUID `json:"session_id,omitempty"`
	Content   string     `json:"content"`
}

func (h *Handler) chat(w http.ResponseWriter, r *http.Request) {
	var in chatWithSessionBody
	if !httpx.Decode(w, r, &in) {
		return
	}

	result, err := h.svc.Chat(in.Content)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.JSON(w, http.StatusCreated, chatWithSessionBody{Content: result})
}

func (h *Handler) chatWithSession(w http.ResponseWriter, r *http.Request) {
	var in chatWithSessionBody
	if !httpx.Decode(w, r, &in) {
		return
	}

	sessID := uuid.Nil
	if in.SessionID != nil {
		sessID = *in.SessionID
	}

	id, result, err := h.svc.ChatWithSession(sessID, in.Content)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.JSON(w, http.StatusCreated, chatWithSessionBody{
		SessionID: &id,
		Content:   result,
	})
}
