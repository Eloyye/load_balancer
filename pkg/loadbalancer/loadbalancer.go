package loadbalancer

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"loadbalancer/pkg/backend"
	"log"
	"net/http"
	"net/url"
)

type LoadBalancer struct {
	http.Handler
	backends []*backend.Backend
	next     int
}

func NewLoadBalancer() *LoadBalancer {
	lb := new(LoadBalancer)
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", lb.rootHandler)
	handler.HandleFunc("/register", lb.registerServerHandler)
	lb.Handler = handler
	lb.backends = nil
	return lb
}

func (l *LoadBalancer) registerServerHandler(writer http.ResponseWriter, request *http.Request) {
	// right now servers can register through post requests
	switch request.Method {
	case http.MethodPost:
		err := l.handleRegisterPOST(writer, request)
		if err != nil {
			// have already handled error to writer
			return
		}
	default:
		writer.WriteHeader(http.StatusBadRequest)
	}

}

func (l *LoadBalancer) handleRegisterPOST(writer http.ResponseWriter, request *http.Request) error {
	var backendResponse backend.BackendDTA
	defer request.Body.Close()
	body, err := io.ReadAll(request.Body)
	if err != nil {
		serveInternalError(writer, fmt.Sprintf("Error reading request body:\n%s", err))
		return err
	}
	err = json.Unmarshal(body, &backendResponse)
	if err != nil {
		serveInternalError(writer, fmt.Sprintf("failed to parse json:\n%s", err))
		return err
	}
	registeredBackend := createBackendInfo(backendResponse)
	l.backends = append(l.backends, registeredBackend)
	log.Printf("successfully logged server at: %s", backendResponse.ServerURL)
	return nil
}

func createBackendInfo(backendResponse backend.BackendDTA) *backend.Backend {
	registeredBackend := new(backend.Backend)
	registeredBackend.Url = backendResponse.ServerURL
	return registeredBackend
}

func (l *LoadBalancer) getNextBackend() (be *backend.Backend, err error) {
	if len(l.backends) < 1 {
		return nil, errors.New("insufficient number of backend servers")
	}
	var backendOut *backend.Backend
	backendOut = l.backends[l.next]
	for backendOut.IsDead {
		l.next = (l.next + 1) % len(l.backends)
		backendOut = l.backends[l.next]
	}
	l.next = (l.next + 1) % len(l.backends)
	return backendOut, nil
}

func (l *LoadBalancer) rootHandler(writer http.ResponseWriter, request *http.Request) {
	printRequestInformation(request)
	//
	firstBackend, err := l.getNextBackend()
	if err != nil {
		serveInternalError(writer, fmt.Sprintf("%q", err))
	}
	fullRequest, err := l.getRequestWithPath(firstBackend)
	if err != nil {
		serveInternalError(writer, fmt.Sprintf("[Error] setup request to %q failed:\n%q", fullRequest, err))
		return
	}
	backendResponse, err := http.Get(fullRequest)
	if err != nil {
		serveInternalError(writer, fmt.Sprintf("[Error] GET request to %q failed:\n%q", fullRequest, err))
		return
	}

	if backendResponse.StatusCode != http.StatusOK {
		serveInternalError(writer, fmt.Sprintf("[Error] GET response yielded incorrect status code %d:\n%q", backendResponse.StatusCode, err))
		return
	}

	defer closeBodyResponse(writer, backendResponse) // Ensure the response body is closed

	err = readResponseBody(writer, backendResponse)
	if err != nil {
		serveInternalError(writer, fmt.Sprintf("[Error] Could not read response body:\n%s", err))
	}
}

func readResponseBody(writer http.ResponseWriter, backendResponse *http.Response) error {
	// Read the response body
	_, err := io.Copy(writer, backendResponse.Body)
	return err
}

func closeBodyResponse(writer http.ResponseWriter, backendResponse *http.Response) {
	func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			serveInternalError(writer, fmt.Sprintf("failed to close body: %s\n", err))
			return
		}
	}(backendResponse.Body)
}

func serveInternalError(writer http.ResponseWriter, message string) {
	http.Error(writer, "Server error", http.StatusInternalServerError)
	log.Print(message)
}

func (l *LoadBalancer) getFirstBackend() *backend.Backend {
	firstBackend := l.backends[0]
	return firstBackend
}

func printRequestInformation(request *http.Request) {
	log.Printf("Received request from %s\n", request.RemoteAddr)
	log.Printf("%s %s %s\n", request.Method, request.RequestURI, request.Proto)
	writeHeaders(request)
}

func (l *LoadBalancer) getRequestWithPath(firstBackend *backend.Backend) (string, error) {
	fullRequest, err := url.JoinPath(firstBackend.Url, "hello")
	fmt.Printf("Made request to %q", fullRequest)
	return fullRequest, err
}

func formatBreak() (int, error) {
	return fmt.Println("===================================")
}

func writeHeaders(request *http.Request) {
	for name, headers := range request.Header {
		for _, h := range headers {
			log.Printf("%v: %v\n", name, h)
		}
	}
}
