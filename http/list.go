package http

import (
	"encoding/json"
	"net/http"

	jstore "github.com/snabble/go-jstore"
)

const ListMaxResults = 1000

func list(store jstore.JStore, provider EntityProvider, withLinks WithLinks, urls *URLBuilder) func(w Response, r Request) {
	toResources := func(items []jstore.Entity) ([]interface{}, error) {
		entities := make([]interface{}, 0, len(items))

		for _, item := range items {
			entity := provider()
			err := json.Unmarshal([]byte(item.JSON), entity)
			if err != nil {
				return []interface{}{}, err
			}

			selfLink := urls.Entity(item.Project, item.DocumentType, item.ID)
			entities = append(entities, withLinks(entity, selfLinks(selfLink)))
		}

		return entities, nil
	}

	return func(w Response, r Request) {
		items, err := store.FindN(r.Project, r.DocumentType, ListMaxResults)
		if err != nil {
			w.SendError(err)
		}
		if len(items) == 0 {
			w.SendError(jstore.NotFound)
		}

		entities, err := toResources(items)
		if err != nil {
			w.SendError(err)
			return
		}

		selfLink := urls.List(r.Project, r.DocumentType)
		w.Send(
			http.StatusOK,
			struct {
				Resources []interface{} `json:"resources"`
				Links     Links         `json:"links"`
			}{
				Resources: entities,
				Links:     selfLinks(selfLink),
			},
		)
	}
}
