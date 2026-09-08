package server

import (
	"net/http"
)

func CheckServer(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ping"))
}
