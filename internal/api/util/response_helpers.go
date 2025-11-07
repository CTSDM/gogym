package util

import (
	"encoding/json"
	"log"
	"net/http"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func RespondWithError(w http.ResponseWriter, code int, msg string, err error) {
	if err != nil {
		log.Println(msg, err.Error())
	}
	if code > 499 {
		log.Printf("Responding with %d error", code)
	}
	RespondWithJSON(w, code, ErrorResponse{
		Error: msg,
	})
}

func RespondWithJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "applilcation/json")
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("error marshalling JSON: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(code)
	if _, err := w.Write(data); err != nil {
		log.Printf("could not write the JSON data: %s", err.Error())
	}
}
