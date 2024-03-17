package test

import (
	"loadbalancer/pkg/backend"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBackend(t *testing.T) {
	server, err := backend.CreateNewBackendServer("", ":8081")
	if err != nil {
		t.Error("did not create backend server")
	}
	request, _ := http.NewRequest(http.MethodGet, "/hello", nil)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	got := response.Body.String()
	want := "hello world"
	assertSameString(t, got, want)
	assertStatusCode(t, response, http.StatusOK)
}
