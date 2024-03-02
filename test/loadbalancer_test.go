package test

import (
	"loadbalancer/pkg/backend"
	"loadbalancer/pkg/loadbalancer"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoadBalancer(t *testing.T) {
	t.Run("test proxy works", func(t *testing.T) {
		backendServer := httptest.NewServer(backend.CreateNewBackendServer())
		defer backendServer.Close()
		backendURL := backendServer.URL
		backends := []*backend.Backend{
			{Url: backendURL},
		}
		server := loadbalancer.NewLoadBalancer(backends)
		request, _ := http.NewRequest(http.MethodGet, "/", nil)
		response := httptest.NewRecorder()
		server.ServeHTTP(response, request)
		got := response.Body.String()
		want := "hello world"
		assertSameString(t, got, want)
		assertStatusCode(t, response, http.StatusOK)
	})
}

func assertStatusCode(t *testing.T, response *httptest.ResponseRecorder, expectedStatusCode int) {
	if response.Code != expectedStatusCode {
		t.Errorf("got status code %d, want status code %d", response.Code, expectedStatusCode)
	}
}

func assertSameString(t *testing.T, got string, want string) {
	t.Helper()
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
