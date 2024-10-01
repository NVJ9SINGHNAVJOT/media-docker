package helper

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-playground/validator/v10"
)

// NOTE: custom validation functions for validate declared below

func customVideoQuality(fl validator.FieldLevel) bool {

	qualityStr := fl.Field().String()
	if qualityStr == "" {
		return true
	}
	quality, err := strconv.Atoi(qualityStr)
	if err != nil {
		return false
	}

	return quality >= 40 && quality <= 100
}

// Declare the validator variable.
var validate *validator.Validate

// Initialize the validator.
func InitializeValidator() {
	validate = validator.New(validator.WithRequiredStructEnabled())
	validate.RegisterValidation("customVideoQuality", customVideoQuality)
}

// ValidateRequestBody validates the request body against the provided struct.
func ValidateRequest(r *http.Request, target interface{}) error {
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		return err
	}

	return validate.Struct(target)
}

// UnmarshalAndValidate takes JSON data in bytes and a pointer to a struct,
// unmarshals the data into the struct and validates it
func UnmarshalAndValidate(data []byte, target interface{}) (string, error) {
	// Unmarshal the incoming data into the target struct
	err := json.Unmarshal(data, target)
	if err != nil {
		return "Failed to unmarshal JSON", err
	}

	// Validate the unmarshalled struct
	if err = validate.Struct(target); err != nil {
		return "Validation failed", err
	}

	// Return success if no errors
	return "Validation and unmarshal succeeded", nil
}
