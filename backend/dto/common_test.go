package dto

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() { gin.SetMode(gin.TestMode) }

func decode(t *testing.T, body []byte) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("invalid JSON body %q: %v", string(body), err)
	}
	return m
}

func TestErrorEnvelopeDerivesCodeAndOmitsErrorsArray(t *testing.T) {
	cases := []struct {
		status   int
		wantCode string
	}{
		{http.StatusBadRequest, CodeBadRequest},
		{http.StatusUnauthorized, CodeUnauthorized},
		{http.StatusNotFound, CodeNotFound},
		{http.StatusConflict, CodeConflict},
		{http.StatusPreconditionFailed, CodePreconditionFailed},
		// A generic 422 (no field details) must NOT claim VALIDATION_ERROR.
		{http.StatusUnprocessableEntity, CodeUnprocessable},
		{http.StatusServiceUnavailable, CodeServiceUnavailable},
		{http.StatusInternalServerError, CodeInternalError},
		{http.StatusTeapot, CodeBadRequest}, // unmapped 4xx falls back to BAD_REQUEST
	}
	for _, tc := range cases {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		Error(c, tc.status, "boom")
		if rec.Code != tc.status {
			t.Fatalf("status = %d, want %d", rec.Code, tc.status)
		}
		body := decode(t, rec.Body.Bytes())
		if body["success"] != false {
			t.Fatalf("success = %v, want false", body["success"])
		}
		if body["code"] != tc.wantCode {
			t.Fatalf("code = %v, want %v", body["code"], tc.wantCode)
		}
		if body["message"] != "boom" {
			t.Fatalf("message = %v, want boom", body["message"])
		}
		// Generic errors must NOT carry an errors array (omitempty).
		if _, ok := body["errors"]; ok {
			t.Fatalf("generic error must omit 'errors', got %v", body["errors"])
		}
	}
}

func TestValidationErrorEnvelope(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	ValidationError(c, ValidationErrors{"name": {"The name field is required."}})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
	body := decode(t, rec.Body.Bytes())
	if body["code"] != CodeValidationError {
		t.Fatalf("code = %v, want %v", body["code"], CodeValidationError)
	}
	if body["message"] != "Validation failed" {
		t.Fatalf("message = %v, want 'Validation failed'", body["message"])
	}
	errs, ok := body["errors"].(map[string]interface{})
	if !ok {
		t.Fatalf("errors must be a field map, got %T", body["errors"])
	}
	if _, ok := errs["name"]; !ok {
		t.Fatalf("expected field error for 'name', got %v", errs)
	}
}

func TestErrorCodeAllowsExplicitCode(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	ErrorCode(c, http.StatusForbidden, "FORBIDDEN_CUSTOM", "no")
	body := decode(t, rec.Body.Bytes())
	if body["code"] != "FORBIDDEN_CUSTOM" {
		t.Fatalf("code = %v, want FORBIDDEN_CUSTOM", body["code"])
	}
}

func TestSuccessEnvelopeCarriesCode(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	OK(c, "ok", gin.H{"a": 1})
	body := decode(t, rec.Body.Bytes())
	if body["success"] != true || body["code"] != CodeOK {
		t.Fatalf("unexpected success envelope: %v", body)
	}
	if _, ok := body["warning"]; ok {
		t.Fatalf("plain success must omit warning, got %v", body["warning"])
	}
}

func TestCreatedAndWarnHelpers(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	Created(c, "made", nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", rec.Code)
	}
	if decode(t, rec.Body.Bytes())["code"] != CodeCreated {
		t.Fatalf("expected CREATED code")
	}

	rec = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(rec)
	OKWarn(c, "saved", gin.H{"id": 1}, "not applied to kernel: boom")
	body := decode(t, rec.Body.Bytes())
	if body["success"] != true || body["code"] != CodeOK {
		t.Fatalf("unexpected warn envelope: %v", body)
	}
	if body["warning"] != "not applied to kernel: boom" {
		t.Fatalf("warning = %v", body["warning"])
	}
}
