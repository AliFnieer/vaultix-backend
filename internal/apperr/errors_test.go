package apperr

import (
	"errors"
	"net/http"
	"testing"
)

func TestNotFound(t *testing.T) {
	err := NotFound("item not found")
	if err.Status != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, err.Status)
	}
	if err.Code != CodeNotFound {
		t.Errorf("Expected code %s, got %s", CodeNotFound, err.Code)
	}
	if err.Message != "item not found" {
		t.Errorf("Expected message %q, got %q", "item not found", err.Message)
	}
}

func TestBadRequest(t *testing.T) {
	err := BadRequest("invalid input")
	if err.Status != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, err.Status)
	}
	if err.Code != CodeValidation {
		t.Errorf("Expected code %s, got %s", CodeValidation, err.Code)
	}
}

func TestConflict(t *testing.T) {
	err := Conflict("already exists")
	if err.Status != http.StatusConflict {
		t.Errorf("Expected status %d, got %d", http.StatusConflict, err.Status)
	}
	if err.Code != CodeConflict {
		t.Errorf("Expected code %s, got %s", CodeConflict, err.Code)
	}
}

func TestUnauthorized(t *testing.T) {
	err := Unauthorized("not logged in")
	if err.Status != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, err.Status)
	}
	if err.Code != CodeUnauthorized {
		t.Errorf("Expected code %s, got %s", CodeUnauthorized, err.Code)
	}
}

func TestForbidden(t *testing.T) {
	err := Forbidden("access denied")
	if err.Status != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, err.Status)
	}
	if err.Code != CodeForbidden {
		t.Errorf("Expected code %s, got %s", CodeForbidden, err.Code)
	}
}

func TestInternal(t *testing.T) {
	err := Internal("something broke")
	if err.Status != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, err.Status)
	}
	if err.Code != CodeInternal {
		t.Errorf("Expected code %s, got %s", CodeInternal, err.Code)
	}
}

func TestWrap(t *testing.T) {
	cause := errors.New("root cause")
	err := Wrap(cause, http.StatusBadRequest, CodeValidation, "wrapped message")
	if err.Status != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, err.Status)
	}
	if err.Code != CodeValidation {
		t.Errorf("Expected code %s, got %s", CodeValidation, err.Code)
	}
	if err.Message != "wrapped message" {
		t.Errorf("Expected message %q, got %q", "wrapped message", err.Message)
	}
	if err.Cause != cause {
		t.Errorf("Expected cause to be %v, got %v", cause, err.Cause)
	}
	if !errors.Is(err, cause) {
		t.Error("errors.Is should find the wrapped cause")
	}
}

func TestAppError_Error(t *testing.T) {
	// Without cause
	err := New(http.StatusNotFound, CodeNotFound, "not found")
	if err.Error() != "not found" {
		t.Errorf("Expected %q, got %q", "not found", err.Error())
	}

	// With cause
	cause := errors.New("db error")
	errWithCause := Wrap(cause, http.StatusInternalServerError, CodeInternal, "query failed")
	expected := "query failed: db error"
	if errWithCause.Error() != expected {
		t.Errorf("Expected %q, got %q", expected, errWithCause.Error())
	}
}

func TestFromError_AppError(t *testing.T) {
	original := NotFound("resource missing")
	result := FromError(original)
	if result != original {
		t.Error("FromError should return the same *AppError if the input is already one")
	}
}

func TestFromError_StandardError(t *testing.T) {
	stdErr := errors.New("some standard error")
	result := FromError(stdErr)
	if result.Status != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, result.Status)
	}
	if result.Code != CodeInternal {
		t.Errorf("Expected code %s, got %s", CodeInternal, result.Code)
	}
	if result.Message != "internal server error" {
		t.Errorf("Expected message %q, got %q", "internal server error", result.Message)
	}
	if result.Cause != stdErr {
		t.Error("Cause should be the original error")
	}
}

func TestFromError_WrappedAppError(t *testing.T) {
	// Use errors.As to unwrap an AppError wrapped by errors.Join or fmt.Errorf
	inner := Conflict("duplicate entry")
	wrapped := errors.Join(errors.New("context"), inner)

	result := FromError(wrapped)
	if result.Code != CodeConflict {
		t.Errorf("Expected code %s, got %s", CodeConflict, result.Code)
	}
}
