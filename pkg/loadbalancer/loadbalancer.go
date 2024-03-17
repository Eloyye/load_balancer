package loadbalancer

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"loadbalancer/pkg/backend"
	"loadbalancer/pkg/utils"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type LoadBalancer struct {
	http.Handler
	backends []*backend.Backend
	next     int
	mutex    sync.Mutex
}

func NewLoadBalancer() *LoadBalancer {
	lb := new(LoadBalancer)
	handler := http.NewServeMux()
	handler.HandleFunc("/hello", lb.rootHandler)
	handler.HandleFunc("/register", lb.registerServerHandler)
	// goroutine that handles health checks
	lb.Handler = handler
	lb.backends = nil
	go func(lb *LoadBalancer) {
		// change this later
		// waiting for some event response to start doing health checks
		// have at least one server before starting the goroutine
		duration := 3 * time.Second
		timeout := 5 * time.Second
		maxTries := 3
		indexDeadServers := []int{}
		t := time.NewTicker(duration)
		for {
			// send for every duration
			for range t.C {
				lb.sendHealthCheckToBackends(timeout, maxTries, indexDeadServers)
			}
		}
	}(lb)
	return lb
}

func (lb *LoadBalancer) sendHealthCheckToBackends(timeout time.Duration, maxTries int, indexDeadServers []int) {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()
	for i, be := range lb.backends {
		go func(be *backend.Backend, i int) {
			be.Mutex.Lock()
			defer be.Mutex.Unlock()
			if be.IsDead && be.ReviveAttempts >= maxTries {
				indexDeadServers = append(indexDeadServers, i)
				return
			}
			toURI, err := url.JoinPath(be.Url, "health")
			if err != nil {
				log.Println(err)
				return
			}

			client := http.Client{
				Timeout: timeout,
			}
			log.Printf("sending health check to %s", be.Url)
			get, err := client.Get(toURI)
			if err != nil {
				log.Println(err)
				be.IsDead = true
				be.ReviveAttempts += 1
				return
			}

			contents, err := io.ReadAll(get.Body)
			defer func(Body io.ReadCloser) {
				err = Body.Close()
				if err != nil {
					log.Println(err)
				}
			}(get.Body)
			if err != nil {
				log.Println(err)
				err = get.Body.Close()
				if err != nil {
					log.Println(err)
				}
				return
			}

			var healthCheckMessage utils.HealthCheckMessage
			err = json.Unmarshal(contents, &healthCheckMessage)
			if err != nil {
				log.Println(err)
				return
			}
			log.Printf("health check success for %s\n", be.Url)
			be.IsDead = false
			be.ReviveAttempts = 0
		}(be, i)
	}
	if len(indexDeadServers) > 0 {
		lb.mutex.Lock()
		for _, i := range indexDeadServers {
			log.Printf("removed %s as valid backends\n", lb.backends[i].Url)
			lb.backends = append(lb.backends[:i], lb.backends[i+1:]...)
		}
		lb.mutex.Unlock()
	}
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
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.backends = append(l.backends, registeredBackend)
	log.Printf("successfully logged server at: %s", backendResponse.ServerURL)
	return nil
}

func createBackendInfo(backendResponse backend.BackendDTA) *backend.Backend {
	registeredBackend := new(backend.Backend)
	registeredBackend.Url = backendResponse.ServerURL
	registeredBackend.ReviveAttempts = 0
	registeredBackend.IsDead = false
	return registeredBackend
}

func (l *LoadBalancer) getNextBackend() (be *backend.Backend, err error) {
	if len(l.backends) < 1 {
		return nil, errors.New("insufficient number of backend servers")
	}
	l.mutex.Lock()
	defer l.mutex.Unlock()
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
	if len(l.backends) < 1 {
		serveInternalError(writer, fmt.Sprintf("There are no backends to serve"))
		return
	}
	firstBackend, err := l.getNextBackend()
	if err != nil {
		serveInternalError(writer, fmt.Sprintf("%q", err))
	}
	fullRequest, err := getRequestWithPath(firstBackend)
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

	firstBackend.Mutex.Lock()
	firstBackend.IsDead = false
	firstBackend.ReviveAttempts = 0
	firstBackend.Mutex.Unlock()

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

func getRequestWithPath(firstBackend *backend.Backend) (string, error) {
	fullRequest, err := url.JoinPath(firstBackend.Url, "hello")
	fmt.Printf("Made request to %q", fullRequest)
	return fullRequest, err
}

func writeHeaders(request *http.Request) {
	for name, headers := range request.Header {
		for _, h := range headers {
			log.Printf("%v: %v\n", name, h)
		}
	}
}
