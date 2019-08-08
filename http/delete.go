package http

import (
	"net/http"
)

func delete(store Store) func(w Response, r Request) {
	return func(w Response, r Request) {
		err := store.Delete(r.EntityID())

		if err != nil {
			w.SendError(err)
			return
		}

		w.Send(http.StatusOK, "")
	}
}
