package httpapi

import (
	"net/http"

	"github.com/shellhaki/spark-whatsapp-module/internal/app"
)

func NewRouter(application *app.App) http.Handler {
	mux := http.NewServeMux()
	handler := NewHandler(application)

	mux.Handle("/", http.RedirectHandler("/ui/", http.StatusTemporaryRedirect))
	mux.HandleFunc("GET /healthz", handler.Health)
	mux.HandleFunc("GET /api/subscribers", handler.ListSubscribers)
	mux.HandleFunc("POST /api/messages/broadcast", handler.SendNotification)
	mux.HandleFunc("POST /api/messages/direct", handler.SendNotificationToPhoneNumber)
	mux.HandleFunc("GET /subscribers", handler.ListSubscribers)
	mux.HandleFunc("POST /notifications/send", handler.SendNotification)
	mux.HandleFunc("POST /notifications/send/direct", handler.SendNotificationToPhoneNumber)
	mux.Handle("/ui/", http.StripPrefix("/ui/", http.FileServer(http.Dir("ui"))))

	return mux
}
