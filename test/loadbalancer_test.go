package test

import (
	"loadbalancer/pkg/backend"
	"loadbalancer/pkg/loadbalancer"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoadBalancer(t *testing.T) {
	t.Run("error when empty backend servers", func(t *testing.T) {
		server := loadbalancer.NewLoadBalancer()
		response := requestHelloWorld(server)
		assertStatusCode(t, response, http.StatusInternalServerError)
	})
	t.Run("creating a backend server will automatically add backend to load balancer", func(t *testing.T) {
		server := httptest.NewServer(loadbalancer.NewLoadBalancer())
		defer server.Close()
		lbURL := server.URL
		backendServer, err := setupBackendServer(lbURL, ":8081")
		if err != nil {
			t.Errorf("%s", err)
		}
		defer backendServer.Close()
	})
}

func requestHelloWorld(server *loadbalancer.LoadBalancer) *httptest.ResponseRecorder {
	request, _ := http.NewRequest(http.MethodGet, "/hello", nil)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	return response
}

func setupBackendServer(lbURL, port string) (*httptest.Server, error) {
	server, err := backend.CreateNewBackendServer(lbURL, port)
	if err != nil {
		return nil, err
	}
	return httptest.NewServer(server), err
}

func assertStatusCode(t *testing.T, response *httptest.ResponseRecorder, expectedStatusCode int) {
	t.Helper()
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
