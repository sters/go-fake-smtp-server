package fakesmtpserver

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
)

// registerListHandlers registers all list-related HTTP endpoints.
func registerListHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/", handleListAllEmails)
}

// handleListAllEmails handles the root endpoint that returns all captured emails.
func handleListAllEmails(w http.ResponseWriter, _ *http.Request) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	err := enc.Encode(sharedBackend.GetAllData())
	if err != nil {
		slog.Info("err", "error", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}
