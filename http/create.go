package http

import (
	"net/http"

	jstore "github.com/snabble/go-jstore"
)

func create(store jstore.JStore,
	extract BodyExtractor,
	withLinks WithLinks,
	urls *URLBuilder,
	cfg config) func(w Response, r Request) {
	return func(w Response, r Request) {
		id, entity, err := extract(r)

		if err != nil {
			w.SendError(err)
			return
		}

		_, err = store.Marshal(entity, jstore.NewID(r.Project, r.DocumentType, id))

		if err != nil {
			w.SendError(err)
			return
		}

		selfLink := urls.Entity(r.Project, r.DocumentType, id)

		w.AddHeader("Location", selfLink.String())

		if cfg.postRespondWithBody {
			w.Send(http.StatusCreated, withLinks(entity, selfLinks(selfLink)))
		} else {
			w.Writer.WriteHeader(http.StatusCreated)
		}
	}
}
