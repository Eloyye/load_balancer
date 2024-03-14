package main

import (
	"fmt"
	"loadbalancer/pkg/loadbalancer"
	"log"
	"net/http"
)

func main() {
	// TODO
	lb := loadbalancer.NewLoadBalancer()
	PORT := ":8080"
	fmt.Printf("Listening on port %s\n", PORT)
	if err := http.ListenAndServe(PORT, lb); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
