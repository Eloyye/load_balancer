package main

import (
	"loadbalancer/pkg/backend"
	"log"
	"net/http"
	"os"
)

func main() {
	argsWithoutProg := os.Args[1:]
	loadBalancerURL := "http://localhost:8080"
	PortNumber := argsWithoutProg[0]
	PORT := ":" + PortNumber
	be := backend.CreateNewBackendServer(loadBalancerURL, PORT)
	if err := http.ListenAndServe(PORT, be); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
