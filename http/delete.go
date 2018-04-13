package http

import (
	"net/http"

	jstore "github.com/snabble/go-jstore"
)

func delete(store jstore.JStore) func(w Response, r Request) {
	return func(w Response, r Request) {
		err := store.Delete(r.EntityID())

		if err != nil {
			w.SendError(err)
			return
		}

		w.Send(http.StatusOK, "")
	}
}
