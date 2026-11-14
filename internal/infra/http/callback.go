package http

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"time"
)

const callbackTimeout = 10 * time.Second

func callback(state string, codeCh chan<- string, errCh chan<- error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if subtle.ConstantTimeCompare([]byte(query.Get("state")), []byte(state)) != 1 {
			http.Error(w, "Invalid state", http.StatusBadRequest)
			return
		}

		if errParam := query.Get("error"); errParam != "" {
			http.Error(w, "Authorization cancel: "+errParam, http.StatusBadRequest)
			notify(errCh, fmt.Errorf("authorization cancel: %s", errParam))
			return
		}

		code := query.Get("code")
		if code == "" {
			http.Error(w, "No code received", http.StatusBadRequest)
			notify(errCh, errors.New("no code received"))
			return
		}

		_, _ = fmt.Fprintln(w, "Authorization received. Can close the browser")
		notify(codeCh, code)
	}
}

func notify[T any](ch chan<- T, value T) {
	select {
	case ch <- value:
	default:
	}
}

func NewCallbackServer(host, state string, codeCh chan<- string, errCh chan<- error) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /callback", callback(state, codeCh, errCh))

	srv := &http.Server{
		Addr:         host,
		Handler:      mux,
		ReadTimeout:  callbackTimeout,
		WriteTimeout: callbackTimeout,
	}
	srv.SetKeepAlivesEnabled(false)
	return srv
}
