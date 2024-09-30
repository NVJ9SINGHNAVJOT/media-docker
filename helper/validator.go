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

// ValidateStruct validates any struct passed in
func ValidateStruct(target interface{}) error {
	return validate.Struct(target)
}
