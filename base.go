package expensefunction

import (
	"encoding/json"
	"net/http"
)

type handlerFunc func(r *http.Request) (interface{}, error)

func baseHandler(hf handlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		resp, err := hf(r)
		if err != nil {
			handleError(w, err)
			return
		}

		// Use json.NewEncoder instead of json.Marshal
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			handleError(w, err)
			return
		}
	}
}

func handleError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	errorResp := map[string]string{"error": err.Error()}
	json.NewEncoder(w).Encode(errorResp)
}
