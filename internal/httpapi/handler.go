package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/shellhaki/spark-whatsapp-module/internal/app"
	"github.com/shellhaki/spark-whatsapp-module/internal/messages"
	"github.com/shellhaki/spark-whatsapp-module/internal/subscribers"
)

type Handler struct {
	app *app.App
}

func NewHandler(application *app.App) *Handler {
	return &Handler{app: application}
}

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) ListSubscribers(w http.ResponseWriter, r *http.Request) {
	items, err := h.app.Subscribers.ListActive(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"subscribers": items})
}

func (h *Handler) SendNotification(w http.ResponseWriter, r *http.Request) {
	var request messages.DeliveryRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	request.Text = strings.TrimSpace(request.Text)

	if err := validateDeliveryRequest(request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	results, err := h.app.Sender.SendToSubscribers(r.Context(), request)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"sent_count": len(results),
		"results":    results,
	})
}

func (h *Handler) SendNotificationToPhoneNumber(w http.ResponseWriter, r *http.Request) {
	var request messages.DirectDeliveryRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	request.PhoneNumber = subscribers.NormalizePhoneNumber(request.PhoneNumber)
	request.Text = strings.TrimSpace(request.Text)

	if err := validateDirectDeliveryRequest(request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.app.Sender.SendToPhoneNumber(r.Context(), request)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, subscribers.ErrSubscriberNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}
