package main

import (
	"net/http"

	"github.com/mattmoor/hello-github/pkg/github"
)

func main() {
	http.HandleFunc("/", github.Handler)
	http.ListenAndServe(":8080", nil)
}
