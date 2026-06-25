package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLiveness(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health/live", nil)

	NewRouter(Dependencies{}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.Status != "ok" {
		t.Fatalf("status body = %q, want %q", response.Status, "ok")
	}
}

func TestSwaggerHTML(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/swagger.html", nil)

	NewRouter(Dependencies{}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != "text/html; charset=utf-8" {
		t.Fatalf("content-type = %q, want text/html; charset=utf-8", contentType)
	}
	if body := recorder.Body.String(); len(body) == 0 || !strings.Contains(body, "SwaggerUIBundle") {
		t.Fatalf("swagger html body does not contain SwaggerUIBundle")
	}
}

func TestOpenAPIYAML(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)

	NewRouter(Dependencies{}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/yaml; charset=utf-8" {
		t.Fatalf("content-type = %q, want application/yaml; charset=utf-8", contentType)
	}
	if body := recorder.Body.String(); !strings.Contains(body, "openapi: 3.0.3") || !strings.Contains(body, "Big Chat Brasil API") {
		t.Fatalf("openapi yaml body does not look like the BCB spec")
	}
}
