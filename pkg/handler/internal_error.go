package handler

import (
	"log"
	"net/http"
)

func InternalError(w http.ResponseWriter, event interface{}, err error) {
	if err != nil {
		log.Printf("Error handling %T: %v", event, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
