package http

import (
	"net/url"

	"github.com/gorilla/mux"
)

type URLBuilder struct {
	router        *mux.Router
	resourceNames map[string]string
}

func NewURLBuilder(router *mux.Router, resourceNames map[string]string) *URLBuilder {
	return &URLBuilder{
		router:        router,
		resourceNames: resourceNames,
	}
}

func (builder *URLBuilder) Entity(project, documentType, id string) *url.URL {
	u, _ := builder.router.
		Get("read").
		URL(
			"project", project,
			"resource", builder.resourceFor(documentType),
			"id", id,
		)
	return u
}

func (builder *URLBuilder) List(project, documentType string) *url.URL {
	u, _ := builder.router.
		Get("list").
		URL(
			"project", project,
			"resource", builder.resourceFor(documentType),
		)
	return u
}

func (builder *URLBuilder) resourceFor(documentType string) string {
	resource, ok := builder.resourceNames[documentType]
	if ok {
		return resource
	}
	return documentType
}
