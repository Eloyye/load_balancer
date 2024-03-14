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

func CreateNewBackendServer(lbURL, port string) (*BackendServer, error) {
	backendServer := new(BackendServer)
	router := http.NewServeMux()
	router.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Reached /hello")
		_, err := fmt.Fprint(w, "hello world")
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
		}
	})
	backendServer.Handler = router
	if lbURL != "" {
		backendServer.loadbalancerURL = lbURL
		backendServer.port = port
		err := backendServer.setup()
		if err != nil {
			return nil, err
		}
	}
	return backendServer, nil
}

type BackendDTA struct {
	ServerURL string
}

func (b *BackendServer) setup() error {
	// Example: Extract server URL (you'll get the actual port here)
	listener, err := net.Listen("tcp", b.port)
	if err != nil {
		return err
	}
	serverURL := "http://" + listener.Addr().String()
	listener.Close() // Close the listener; we're just getting the address
	backendDta := BackendDTA{ServerURL: serverURL}
	json, err := json2.Marshal(backendDta)
	if err != nil {
		return err
	}
	log.Println("Server URL:", serverURL)
	b.ServerURL = serverURL

	pathToRegister, err := url.JoinPath(b.loadbalancerURL, "register")
	if err != nil {
		return err
	}
	log.Printf("sent register request to %s", pathToRegister)
	response, err := http.Post(pathToRegister, "application/json", bytes.NewReader(json))
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusOK {
		return err
	}
	log.Println("Successfully registered to load balancer")
	return nil
}
