package fakesmtpserver

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
)

// registerSearchHandlers registers all search-related HTTP endpoints.
func registerSearchHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/search/to", handleSearchEndpoint(FieldTo))
	mux.HandleFunc("/search/cc", handleSearchEndpoint(FieldCC))
	mux.HandleFunc("/search/bcc", handleSearchEndpoint(FieldBCC))
	mux.HandleFunc("/search/from", handleSearchEndpoint(FieldFrom))
}

// handleSearchEndpoint returns a handler function for the specified search field.
func handleSearchEndpoint(field string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only allow GET requests
		if r.Method != http.MethodGet {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")

			return
		}

		// Parse and validate email parameter
		email, err := parseEmailParameter(r.URL.Query())
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())

			return
		}

		// Perform search
		results, err := sharedBackend.SearchByField(field, email)
		if err != nil {
			slog.Info("search error", "field", field, "email", email, "error", err)
			writeJSONError(w, http.StatusInternalServerError, "search failed")

			return
		}

		// Return results as JSON
		buf := &bytes.Buffer{}
		enc := json.NewEncoder(buf)
		if err := enc.Encode(results); err != nil {
			slog.Info("encoding error", "error", err)
			writeJSONError(w, http.StatusInternalServerError, "encoding failed")

			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(buf.Bytes())
	}
}
