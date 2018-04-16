package http

import (
	"net/url"

	"github.com/gorilla/mux"
)

type URLBuilder struct {
	router *mux.Router
}

func NewURLBuilder(router *mux.Router) *URLBuilder {
	return &URLBuilder{router: router}
}

func (builder *URLBuilder) Entity(project, documentType, id string) *url.URL {
	u, _ := builder.router.
		Get("read").
		URL(
			"project", project,
			"documentType", documentType,
			"id", id,
		)
	return u
}

func (builder *URLBuilder) List(project, documentType string) *url.URL {
	u, _ := builder.router.
		Get("list").
		URL(
			"project", project,
			"documentType", documentType,
		)
	return u
}
