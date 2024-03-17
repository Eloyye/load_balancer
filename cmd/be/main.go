package main

import (
	"loadbalancer/pkg/backend"
	"loadbalancer/pkg/utils"
	"log"
	"net/http"
	"os"
)

func main() {
	argsWithoutProg := os.Args[1:]
	loadBalancerURL := "http://localhost:8080"

	var PortNumber string

	if len(argsWithoutProg) < 1 {
		PortNumber = "80"
	} else {
		PortNumber = argsWithoutProg[0]
	}

	PORT := utils.FormatPort(PortNumber)

	// initialize backend server
	be, err := backend.CreateNewBackendServer(loadBalancerURL, PORT)
	if err != nil {
		failStartup(err)
	}

	// listen and startup backend http server
	err = http.ListenAndServe(PORT, be)
	if err != nil {
		failStartup(err)
	}
}

func failStartup(err error) {
	log.Fatalf("Failed to start server: %v\n", err)
}
