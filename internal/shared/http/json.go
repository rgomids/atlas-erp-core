package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func DecodeJSON(request *http.Request, target any) error {
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode json: %w", err)
	}

	if decoder.More() {
		return fmt.Errorf("decode json: unexpected trailing data")
	}

	return nil
}

func WriteJSON(writer http.ResponseWriter, statusCode int, payload any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(statusCode)
	_ = json.NewEncoder(writer).Encode(payload)
}
