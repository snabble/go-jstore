package http

import (
	"net/http"

	jstore "github.com/snabble/go-jstore"
)

func get(store Store, provider EntityProvider, withLinks WithLinks, urls *URLBuilder) func(w Response, r Request) {
	return func(w Response, r Request) {
		var entity interface{}
		entity = provider()

		err := store.Unmarshal(entity, r.Project, r.DocumentType, jstore.Id(r.ID))

		if err != nil {
			w.SendError(err)
		} else {
			selfLink := urls.Entity(r.Project, r.DocumentType, r.ID)
			w.Send(http.StatusOK, withLinks(entity, selfLinks(selfLink)))
		}
	}
}
