package backend

import (
	"fmt"
	"net/http"
)

type BackendServer struct {
	http.Handler
}

func CreateNewBackendServer() *BackendServer {
	backendServer := new(BackendServer)
	router := http.NewServeMux()
	router.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "hello world")
	})
	backendServer.Handler = router
	return backendServer
}
