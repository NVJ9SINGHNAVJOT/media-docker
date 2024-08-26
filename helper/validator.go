package helper

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"
)

// Initialize the validator.
var validate = validator.New(validator.WithRequiredStructEnabled())

// ValidateRequestBody validates the request body against the provided struct.
func ValidateRequest(r *http.Request, target interface{}) error {
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		return err
	}

	if err := validate.Struct(target); err != nil {
		return err
	}

	return nil
}
