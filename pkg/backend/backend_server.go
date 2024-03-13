package backend

import (
	"bytes"
	json2 "encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
)

type BackendServer struct {
	loadbalancerURL string
	ServerURL       string
	port            string
	http.Handler
}

func CreateNewBackendServer(lbURL, port string) *BackendServer {
	backendServer := new(BackendServer)
	router := http.NewServeMux()
	router.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "hello world")
	})
	backendServer.Handler = router
	if lbURL != "" {
		backendServer.loadbalancerURL = lbURL
		backendServer.port = port
		backendServer.setup()
	}
	return backendServer
}

type BackendDTA struct {
	ServerURL string
}

func (b *BackendServer) setup() {
	// Example: Extract server URL (you'll get the actual port here)
	listener, err := net.Listen("tcp", b.port)
	if err != nil {
		log.Fatal("Error getting server address:", err)
		return
	}
	serverURL := "http://" + listener.Addr().String()
	listener.Close() // Close the listener; we're just getting the address
	backendDta := BackendDTA{ServerURL: serverURL}
	json, err := json2.Marshal(backendDta)
	if err != nil {
		log.Fatal("Error marshalling json:", err)
		return
	}
	log.Println("Server URL:", serverURL)
	b.ServerURL = serverURL

	pathToRegister, err := url.JoinPath(b.loadbalancerURL, "register")
	if err != nil {
		log.Fatalf("Error with joining path:", err)
		return
	}
	log.Printf("sent register request to %s", pathToRegister)
	response, err := http.Post(pathToRegister, "application/json", bytes.NewReader(json))
	if err != nil {
		log.Fatalf("Error with POST request to Load Balancer:", err)
		return
	}
	if response.StatusCode != http.StatusOK {
		log.Fatalf("Error, expected status code %d response but got status code %d:\n%s", http.StatusOK, response.StatusCode, err)
		return
	}
	log.Println("Successfully registered to load balancer")
}
