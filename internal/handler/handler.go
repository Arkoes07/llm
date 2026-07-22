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
	grogapiSvc service.Service
}

func New(grogapiSvc service.Service) *Handler {
	return &Handler{
		grogapiSvc: grogapiSvc,
	}
}
func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Timeout(requestTimeout))

	r.Post("/ot-chat", h.chatWithoutSession)
	r.Post("/chat", h.chat)
	r.Post("/agent-chat", h.agentChat)

	return r
}

type chatBody struct {
	SessionID *uuid.UUID `json:"session_id,omitempty"`
	Content   string     `json:"content"`
}

func (h *Handler) chatWithoutSession(w http.ResponseWriter, r *http.Request) {
	var in chatBody
	if !httpx.Decode(w, r, &in) {
		return
	}

	result, err := h.grogapiSvc.ChatWithoutSession(in.Content)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.JSON(w, http.StatusCreated, chatBody{Content: result})
}

func (h *Handler) chat(w http.ResponseWriter, r *http.Request) {
	var in chatBody
	if !httpx.Decode(w, r, &in) {
		return
	}

	sessID := uuid.Nil
	if in.SessionID != nil {
		sessID = *in.SessionID
	}

	id, result, err := h.grogapiSvc.Chat(sessID, in.Content)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.JSON(w, http.StatusCreated, chatBody{
		SessionID: &id,
		Content:   result,
	})
}

func (h *Handler) agentChat(w http.ResponseWriter, r *http.Request) {
	var in chatBody
	if !httpx.Decode(w, r, &in) {
		return
	}

	sessID := uuid.Nil
	if in.SessionID != nil {
		sessID = *in.SessionID
	}

	id, result, err := h.grogapiSvc.AgentChat(sessID, in.Content)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.JSON(w, http.StatusCreated, chatBody{
		SessionID: &id,
		Content:   result,
	})
}
