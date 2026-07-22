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
	r.Post("/chats", h.createChat)
	r.Post("/chats/v2", h.createChatWithSession)

	return r
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type createChatWithSessionBody struct {
	SessionID *uuid.UUID `json:"session_id,omitempty"`
	Content   string     `json:"content"`
}

func (h *Handler) createChat(w http.ResponseWriter, r *http.Request) {
	var in createChatWithSessionBody
	if !httpx.Decode(w, r, &in) {
		return
	}

	result, err := h.svc.CreateChat(in.Content)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.JSON(w, http.StatusCreated, createChatWithSessionBody{Content: result})
}

func (h *Handler) createChatWithSession(w http.ResponseWriter, r *http.Request) {
	var in createChatWithSessionBody
	if !httpx.Decode(w, r, &in) {
		return
	}

	sessID := uuid.Nil
	if in.SessionID != nil {
		sessID = *in.SessionID
	}

	id, result, err := h.svc.CreateChatWithSession(sessID, in.Content)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.JSON(w, http.StatusCreated, createChatWithSessionBody{
		SessionID: &id,
		Content:   result,
	})
}
