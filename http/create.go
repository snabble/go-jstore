package http

import (
	"net/http"

	jstore "github.com/snabble/go-jstore"
)

func create(store jstore.JStore, extract BodyExtractor, withLinks WithLinks, urls *URLBuilder) func(w Response, r Request) {
	return func(w Response, r Request) {
		id, entity, err := extract(r)

		if err != nil {
			w.SendError(err)
			return
		}

		err = store.Marshal(entity, r.Project, r.DocumentType, id)

		if err != nil {
			w.SendError(err)
			return
		}

		selfLink := urls.Entity(r.Project, r.DocumentType, id)
		w.AddHeader("Location", selfLink.String())
		w.Send(http.StatusCreated, withLinks(entity, selfLinks(selfLink)))
	}
}
