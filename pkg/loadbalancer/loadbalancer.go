package loadbalancer

import (
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
}

func NewLoadBalancer(backends []*backend.Backend) *LoadBalancer {
	lb := new(LoadBalancer)
	handler := http.NewServeMux()
	handler.HandleFunc("/", lb.rootHandler)
	handler.HandleFunc("/register", lb.registerServerHandler)
	lb.Handler = handler
	lb.backends = backends
	return lb
}

func (l *LoadBalancer) registerServerHandler(writer http.ResponseWriter, request *http.Request) {
	// right now servers can register through post requests
	switch request.Method {
	case http.MethodPost:

	default:
		writer.WriteHeader(http.StatusBadRequest)
	}

}

func (l *LoadBalancer) rootHandler(writer http.ResponseWriter, request *http.Request) {
	printRequestInformation(request)
	//
	if len(l.backends) < 1 {
		serveInternalError(writer, fmt.Sprintf("[Error] Insufficient servers registered"))
		return
	}
	firstBackend := l.getFirstBackend()
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
		serveInternalError(writer, fmt.Sprintf("[Error] GET response from %q yielded incorrect status code %d:\n%q", backendResponse, backendResponse.StatusCode, err))
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
