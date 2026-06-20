package middleware

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/example/vpn-manager/dto"
	"github.com/go-playground/validator/v10"
)

// validate is a shared validator instance.
var validate = validator.New()

func init() {
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" || name == "" {
			return fld.Name
		}
		return name
	})
}

// Validate runs struct validation and returns Laravel-style field errors:
//   {"field_name": ["message"]}
// Field names are taken from json tags so React Hook Form can call
// setError("field_name", ...) without extra mapping.
func Validate(payload interface{}) dto.ValidationErrors {
	err := validate.Struct(payload)
	if err == nil {
		return nil
	}

	errs := make(dto.ValidationErrors)
	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		errs["_"] = []string{err.Error()}
		return errs
	}

	for _, fe := range validationErrors {
		field := fe.Field()
		errs[field] = append(errs[field], messageForTag(fe))
	}
	return errs
}

func messageForTag(fe validator.FieldError) string {
	field := strings.ReplaceAll(fe.Field(), "_", " ")
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("The %s field is required.", field)
	case "min":
		return fmt.Sprintf("The %s must be at least %s.", field, fe.Param())
	case "max":
		return fmt.Sprintf("The %s must be at most %s.", field, fe.Param())
	case "gt":
		return fmt.Sprintf("The %s must be greater than %s.", field, fe.Param())
	case "gte":
		return fmt.Sprintf("The %s must be greater than or equal to %s.", field, fe.Param())
	case "cidr":
		return fmt.Sprintf("The %s must be a valid CIDR address.", field)
	case "ip":
		return fmt.Sprintf("The %s must be a valid IP address.", field)
	case "email":
		return fmt.Sprintf("The %s must be a valid email address.", field)
	default:
		return fmt.Sprintf("The %s failed validation on '%s'.", field, fe.Tag())
	}
}
