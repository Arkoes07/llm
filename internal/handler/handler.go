package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/arkoes07/llm/internal/service"
)

const requestTimeout = 30 * time.Second

type Handler struct {
	groqrawapiSvc service.Service
	groqlibSvc    service.Service
}

func New(groqrawapiSvc service.Service, groqlibSvc service.Service) *Handler {
	return &Handler{
		groqrawapiSvc: groqrawapiSvc,
		groqlibSvc:    groqlibSvc,
	}
}
func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Timeout(requestTimeout))

	r.Post("/chat/no-memory", h.chatWithoutSession)
	r.Post("/chat", h.chat)
	r.Post("/chat/agent/weather", h.weatherAgentChat)
	r.Post("/chat/agent/log-triage", h.logTriageAgentChat)

	return r
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	log.Println("request failed", "err", msg)
	writeJSON(w, status, map[string]string{"error": msg})
}

func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return false
	}
	return true
}

type chatBody struct {
	SessionID      uuid.UUID `json:"session_id,omitzero"`
	Content        string    `json:"content"`
	Implementation string    `json:"implementation"`
}

func (h *Handler) chatWithoutSession(w http.ResponseWriter, r *http.Request) {
	var in chatBody
	if !decode(w, r, &in) {
		return
	}

	result, err := h.getSvc(in.Implementation).ChatWithoutSession(in.Content)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, chatBody{Content: result})
}

func (h *Handler) chat(w http.ResponseWriter, r *http.Request) {
	var in chatBody
	if !decode(w, r, &in) {
		return
	}

	id, result, err := h.getSvc(in.Implementation).Chat(in.SessionID, in.Content)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, chatBody{
		SessionID: id,
		Content:   result,
	})
}

func (h *Handler) weatherAgentChat(w http.ResponseWriter, r *http.Request) {
	var in chatBody
	if !decode(w, r, &in) {
		return
	}

	id, result, err := h.getSvc(in.Implementation).AgentChat("weather", in.SessionID, in.Content)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, chatBody{
		SessionID: id,
		Content:   result,
	})
}

func (h *Handler) logTriageAgentChat(w http.ResponseWriter, r *http.Request) {
	var in chatBody
	if !decode(w, r, &in) {
		return
	}

	id, result, err := h.getSvc(in.Implementation).AgentChat("log_triage", in.SessionID, in.Content)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, chatBody{
		SessionID: id,
		Content:   result,
	})
}

func (h *Handler) getSvc(name string) service.Service {
	if name == "groq_raw_api" {
		return h.groqrawapiSvc
	}
	return h.groqlibSvc
}
