package http

import (
	"errors"
	"fmt"
	"net/http"
	"time"
)

func callback(codeCh chan<- string, errCh chan<- error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if errParam := r.URL.Query().Get("error"); errParam != "" {
			http.Error(w, "Authorization cancel: "+errParam, http.StatusBadRequest)
			select {
			case errCh <- fmt.Errorf("authorization cancel: %s", errParam):
			default:
			}
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "No code received", http.StatusBadRequest)
			select {
			case errCh <- errors.New("no code received"):
			default:
			}
			return
		}

		_, _ = fmt.Fprintln(w, "Authorization received. Can close the browser")

		select {
		case codeCh <- code:
		default:
		}
	}
}

func NewCallbackServer(host string, codeCh chan<- string, errCh chan<- error) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", callback(codeCh, errCh))

	return &http.Server{
		Addr:         host,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
}
