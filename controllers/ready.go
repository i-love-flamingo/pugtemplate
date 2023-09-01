package controllers

import (
	"net/http"

	"flamingo.me/pugtemplate/pugjs"
)

type (
	Ready struct {
		startup *pugjs.Startup
	}
)

// Inject dependencies
func (r *Ready) Inject(
	startup *pugjs.Startup,
) *Ready {
	r.startup = startup

	return r
}

// ServeHTTP responds to PugJS Ready requests
func (r *Ready) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	if r.startup.IsFinished() {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("All pugjs startup tasks are finished"))
		return
	}

	w.WriteHeader(http.StatusTooEarly)
	_, _ = w.Write([]byte("Still waiting for pugjs startup tasks to be finished"))
}
