package ingest

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"watchtrail/internal/event"
)

const maxBodyBytes = 1 << 20 // 1 MiB cap on a single request

// HTTPHandler returns the /ingest handler. token is the expected bearer value;
// an empty token disables auth (loopback-only deployments).
func (p *Pipeline) HTTPHandler(token string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /ingest", func(w http.ResponseWriter, r *http.Request) {
		if !authorized(r, token) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBodyBytes))
		if err != nil {
			http.Error(w, "read error", http.StatusBadRequest)
			return
		}

		items, err := splitBatch(body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Events are processed in order; on the first failure we stop and return
		// the matching status. Already-processed events in this batch stay
		// committed — that is safe because ingestion is idempotent by event_id,
		// so a client that retries the whole batch will not double-count.
		for _, raw := range items {
			if err := p.Process(r.Context(), raw); err != nil {
				if errors.Is(err, event.ErrValidation) || errors.Is(err, event.ErrUnsupportedVersion) {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
		}

		if len(items) == 1 && body[firstNonSpace(body)] != '[' {
			w.WriteHeader(http.StatusAccepted) // single event
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]int{"accepted": len(items)})
	})
	return mux
}

func authorized(r *http.Request, token string) bool {
	if token == "" {
		return true
	}
	const prefix = "Bearer "
	h := r.Header.Get("Authorization")
	if len(h) <= len(prefix) || h[:len(prefix)] != prefix {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(h[len(prefix):]), []byte(token)) == 1
}

// splitBatch returns one raw-JSON slice per event, handling both a single object
// and a top-level array. Each element's bytes are preserved for raw storage.
func splitBatch(body []byte) ([][]byte, error) {
	i := firstNonSpace(body)
	if i < 0 {
		return nil, errors.New("empty body")
	}
	if body[i] == '[' {
		var arr []json.RawMessage
		if err := json.Unmarshal(body, &arr); err != nil {
			return nil, errors.New("invalid JSON array")
		}
		out := make([][]byte, len(arr))
		for j, m := range arr {
			out[j] = m
		}
		return out, nil
	}
	return [][]byte{body}, nil
}

func firstNonSpace(b []byte) int {
	for i, c := range b {
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			return i
		}
	}
	return -1
}
