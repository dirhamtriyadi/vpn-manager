package middleware

import (
	"fmt"

	"github.com/go-playground/validator/v10"
)

// validate is a shared validator instance.
var validate = validator.New()

// Validate runs struct validation and returns a map of field -> message
// suitable for returning in a JSON error response. It returns nil when valid.
func Validate(payload interface{}) map[string]string {
	err := validate.Struct(payload)
	if err == nil {
		return nil
	}

	errs := make(map[string]string)
	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		errs["_"] = err.Error()
		return errs
	}

	for _, fe := range validationErrors {
		errs[fe.Field()] = messageForTag(fe)
	}
	return errs
}

func messageForTag(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "this field is required"
	case "min":
		return fmt.Sprintf("must be at least %s", fe.Param())
	case "max":
		return fmt.Sprintf("must be at most %s", fe.Param())
	case "gt":
		return fmt.Sprintf("must be greater than %s", fe.Param())
	case "gte":
		return fmt.Sprintf("must be greater than or equal to %s", fe.Param())
	case "email":
		return "must be a valid email address"
	default:
		return fmt.Sprintf("failed validation on '%s'", fe.Tag())
	}
}
