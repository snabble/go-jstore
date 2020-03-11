package http

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	jstore "github.com/snabble/go-jstore/v2"
	logging "github.com/snabble/go-logging/v2"
)

type Permit func(request Request) bool
type QueryExtractor func(r Request) (limit int, query []jstore.Option, err error)
type BodyExtractor func(r Request) (string, interface{}, error)
type EntityProvider func() interface{}
type WithLinks func(entity interface{}, links Links) interface{}

type Store interface {
	Marshal(object interface{}, id jstore.EntityID) (jstore.EntityID, error)
	Unmarshal(entityOrObjectRef interface{}, project, documentType string, options ...jstore.Option) error
	Delete(id jstore.EntityID) error
	FindN(project, documentType string, maxResults int, options ...jstore.Option) ([]jstore.Entity, error)
}

func Expose(
	router *mux.Router,
	store Store,
	canCreate Permit,
	canRead Permit,
	canUpdate Permit,
	canDelete Permit,
	queryExtractor QueryExtractor,
	bodyExtractor BodyExtractor,
	provider EntityProvider,
	withLinks WithLinks,
	allowedDocumentTypes []string,
	resourceNames map[string]string,
	configOpts ...ConfigOption,
) *mux.Router {

	register := func(name, path, method string, permit Permit, handler func(w Response, r Request)) {
		router.Handle(path,
			createHandler(
				permit,
				handler,
				allowedDocumentTypes,
				resourceNames,
			),
		).
			Methods(method).
			Name(name)
	}

	urls := NewURLBuilder(router, resourceNames)

	cfg := configFromOptions(configOpts)

	register("create", "/{project}/{resource}", http.MethodPost, canCreate, create(store, bodyExtractor, withLinks, urls, cfg))
	register("read", "/{project}/{resource}/{id}", http.MethodGet, canRead, get(store, provider, withLinks, urls))
	register("list", "/{project}/{resource}", http.MethodGet, canRead, list(store, provider, queryExtractor, withLinks, urls))
	register("update", "/{project}/{resource}/{id}", http.MethodPut, canUpdate, update(store, bodyExtractor, withLinks, urls))
	register("delete", "/{project}/{resource}/{id}", http.MethodDelete, canDelete, delete(store))

	return router
}

func createHandler(
	permit Permit,
	handler func(w Response, r Request),
	allowedDocumentTypes []string,
	resourceNames map[string]string,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		project, ok := verifyProject(w, r)
		if !ok {
			return
		}
		resource, ok := verifyResource(w, r)
		if !ok {
			return
		}
		documentType := mapResourceToDocumentType(resourceNames, resource)

		id := mux.Vars(r)["id"]

		if !contains(allowedDocumentTypes, documentType) {
			sendError(w, errors.New("not found"), http.StatusNotFound)
			return
		}

		request := Request{OriginalRequest: r, Project: project, DocumentType: documentType, ID: id}
		if !permit(request) {
			sendError(w, errors.New("forbidden"), http.StatusForbidden)
			return
		}
		handler(Response{Writer: w}, request)
	})
}

// verifyProject checks, that a parameter with the name 'project'
// exists.  Otherwise, it send an error to the client
func verifyProject(w http.ResponseWriter, r *http.Request) (string, bool) {
	t, exist := mux.Vars(r)["project"]

	if !exist || t == "" {
		sendError(w, errors.New("project parameter missing"), http.StatusBadRequest)
		return "", false
	}

	return t, true
}

// verifyResource checks, that a parameter with the name
// 'resource' exists. Otherwise, it send an error to the client
func verifyResource(w http.ResponseWriter, r *http.Request) (string, bool) {
	t, exist := mux.Vars(r)["resource"]

	if !exist || t == "" {
		sendError(w, errors.New("not found"), http.StatusNotFound)
		return "", false
	}

	return t, true
}

// mapResourceToDocumentType selects the documentType associated with
// the resource
func mapResourceToDocumentType(mapping map[string]string, param string) string {
	for documentType, resource := range mapping {
		if resource == param {
			return documentType
		}
	}
	return param
}

func sendError(w http.ResponseWriter, err error, status int) bool {
	w.WriteHeader(status)
	if status >= 500 {
		// Do not let internal messages to the user
		fmt.Fprintln(w, "Internal Server Error")
		logging.Log.WithError(err).Errorf("Internal Server Error, Statuscode: %v", status)
	} else {
		fmt.Fprintln(w, err)
		logging.Log.WithError(err).Warnf("Client Error, Statuscode: %v", status)
	}
	return true
}

func contains(arr []string, search string) bool {
	for _, s := range arr {
		if s == search {
			return true
		}
	}
	return false
}
