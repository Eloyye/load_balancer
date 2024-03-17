package test

import (
	"io"
	"loadbalancer/pkg/backend"
	"loadbalancer/pkg/loadbalancer"
	"loadbalancer/pkg/utils"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestIntegration(t *testing.T) {
	t.Run("one load balancer, two backends, send /hello to lb", func(t *testing.T) {
		lb := loadbalancer.NewLoadBalancer()
		lbServer := httptest.NewServer(lb)
		lbURL := lbServer.URL
		defer lbServer.Close()
		be1, err := backend.CreateNewBackendServer(lbURL, utils.FormatPort("80"))
		if err != nil {
			t.Errorf("failed to create a new backend server:\n%s\n", err)
			return
		}
		be2, err := backend.CreateNewBackendServer(lbURL, utils.FormatPort("81"))
		if err != nil {
			t.Errorf("failed to create a new backend server:\n%s\n", err)
			return
		}
		be1server := httptest.NewServer(be1)
		defer be1server.Close()
		be2server := httptest.NewServer(be2)
		defer be2server.Close()
		result, err := url.JoinPath(lbServer.URL, "hello")
		if err != nil {
			t.Errorf("failed to join path:\n%s\n", err)
			return
		}
		response, err := http.Get(result)
		if err != nil {
			t.Errorf("failed to get response:\n%s\n", err)
			return
		}
		if response.StatusCode != http.StatusOK {
			t.Errorf("Incorrect status code, got %d but want %d:\n", response.StatusCode, http.StatusOK)
			return
		}
		defer response.Body.Close()
		gotBytes, err := io.ReadAll(response.Body)
		if err != nil {
			t.Errorf("could not read response body:\n%s\n", err)
			return
		}
		got := string(gotBytes[:])
		want := "hello world"
		if got != want {
			t.Errorf("got %q, want %q\n", got, want)
		}
	})
}
