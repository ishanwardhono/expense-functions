package expensefunction

import (
	"encoding/json"
	"net/http"
)

type handlerFunc func(r *http.Request) (interface{}, error)

func baseHandler(hf handlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

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
