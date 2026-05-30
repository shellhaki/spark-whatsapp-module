package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/shellhaki/spark-whatsapp-module/internal/messages"
)

func validateDeliveryRequest(request messages.DeliveryRequest) error {
	if strings.TrimSpace(request.Text) == "" {
		return errors.New("text is required")
	}

	return nil
}

func validateDirectDeliveryRequest(request messages.DirectDeliveryRequest) error {
	if strings.TrimSpace(request.PhoneNumber) == "" {
		return errors.New("phone_number is required")
	}

	return validateDeliveryRequest(messages.DeliveryRequest{
		Text: request.Text,
	})
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
