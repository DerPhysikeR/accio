package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestQueryHandlerReturnsString(t *testing.T) {
	req := httptest.NewRequest("GET", "/hello", nil)
	w := httptest.NewRecorder()

	createQueryHandler(func() string { return "test" })(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	body := w.Body.String()
	expected := "test"
	if body != expected {
		t.Fatalf("expected body %q, got %q", expected, body)
	}
}
