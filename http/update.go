package http

import (
	"net/http"
)

func update(store Store, extract BodyExtractor, withLinks WithLinks, urls *URLBuilder) func(w Response, r Request) {
	return func(w Response, r Request) {
		id, entity, err := extract(r)

		if err != nil {
			w.SendError(err)
			return
		}

		if id != "" && id != r.ID {
			w.SendError(ClientError("invalid id"))
			return
		}

		_, err = store.Marshal(&entity, r.EntityID())

		if err != nil {
			w.SendError(err)
			return
		}
		selfLink := urls.Entity(r.Project, r.DocumentType, r.ID)
		w.Send(http.StatusOK, withLinks(entity, selfLinks(selfLink)))
	}
}
