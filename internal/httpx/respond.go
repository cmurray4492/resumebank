package httpx

import "net/http"

func NotFound(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not Found", http.StatusNotFound)
}

func BadRequest(w http.ResponseWriter, message string) {
	http.Error(w, message, http.StatusBadRequest)
}

func ServerError(w http.ResponseWriter, err error) {
	http.Error(w, "Internal Server Error: "+err.Error(), http.StatusInternalServerError)
}
