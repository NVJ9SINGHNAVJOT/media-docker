package helper

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strconv"

	"github.com/go-playground/validator/v10"
)

// NOTE: custom validation functions for validate declared below

// customVideoQuality is a custom validation function that checks the validity of video quality.
// It ensures that the quality value, which is expected as a string, is within a range of 40 to 100.
// If the string is empty, the function returns true, meaning the field passes validation.
// Otherwise, it converts the string to an integer and checks if it falls within the valid range.
func customVideoQuality(fl validator.FieldLevel) bool {

	qualityStr := fl.Field().String() // Get the field value as a string
	if qualityStr == "" {
		return true // Allow empty strings to pass validation
	}
	quality, err := strconv.Atoi(qualityStr) // Convert the string to an integer
	if err != nil {
		return false // If conversion fails, the validation fails
	}

	return quality >= 40 && quality <= 100 // Return true if the quality is between 40 and 100
}

// customNonNegativeInt is a custom validation function for integer fields.
// It ensures that the field is either an int or int64, and that the value is non-negative (0 or positive).
// This allows zero to be a valid value but rejects negative integers.
func customNonNegativeInt(fl validator.FieldLevel) bool {
	// Check if the field is an int or int64 type
	if fl.Field().Kind() == reflect.Int || fl.Field().Kind() == reflect.Int64 {
		value := fl.Field().Int() // Get the integer value
		return value >= 0         // Return true if the value is 0 or positive
	}
	return false // If the field is not an integer, the validation fails
}

// Declare the validator variable.
// This global variable is used to perform validation on structs.
var validate *validator.Validate

// InitializeValidator initializes the validator and registers custom validation functions.
// It registers the 'customVideoQuality' and 'customNonNegativeInt' validators
// to allow custom validation logic for specific fields in your structs.
func InitializeValidator() {
	validate = validator.New(validator.WithRequiredStructEnabled()) // Enable validation of required struct fields

	// Register custom validators
	validate.RegisterValidation("customVideoQuality", customVideoQuality)     // Register video quality validator
	validate.RegisterValidation("customNonNegativeInt", customNonNegativeInt) // Register non-negative integer validator
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
