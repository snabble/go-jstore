package http

import (
	"net/http"

	jstore "github.com/snabble/go-jstore"
)

func update(store jstore.JStore, extract BodyExtractor, withLinks WithLinks, urls *URLBuilder) func(w Response, r Request) {
	return func(w Response, r Request) {
		id, entity, err := extract(r)

		if err != nil {
			w.SendError(err)
			return
		}

		if id != r.ID {
			w.SendError(ClientError("invalid id"))
			return
		}

		err = store.Marshal(&entity, r.Project, r.DocumentType, r.ID)

		if err != nil {
			w.SendError(err)
			return
		}
		selfLink := urls.Entity(r.Project, r.DocumentType, r.ID)
		w.Send(http.StatusOK, withLinks(entity, selfLinks(selfLink)))
	}
}
