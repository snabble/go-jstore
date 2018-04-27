package http

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	jstore "github.com/snabble/go-jstore"
)

type Permit func(request Request) bool
type QueryExtractor func(r Request) (limit int, query []jstore.Option, err error)
type BodyExtractor func(r Request) (string, interface{}, error)
type EntityProvider func() interface{}
type WithLinks func(entity interface{}, links Links) interface{}

func Expose(
	router *mux.Router,
	store jstore.JStore,
	canCreate Permit,
	canRead Permit,
	canUpdate Permit,
	canDelete Permit,
	queryExtractor QueryExtractor,
	bodyExtractor BodyExtractor,
	provider EntityProvider,
	withLinks WithLinks,
	allowedDocumentTypes []string,
) *mux.Router {

	register := func(name, path, method string, permit Permit, handler func(w Response, r Request)) {
		router.Handle(path,
			createHandler(
				permit,
				handler,
				allowedDocumentTypes,
			),
		).
			Methods(method).
			Name(name)
	}

	urls := NewURLBuilder(router)

	register("create", "/{project}/{documentType}", http.MethodPost, canCreate, create(store, bodyExtractor, withLinks, urls))
	register("read", "/{project}/{documentType}/{id}", http.MethodGet, canRead, get(store, provider, withLinks, urls))
	register("list", "/{project}/{documentType}", http.MethodGet, canRead, list(store, provider, queryExtractor, withLinks, urls))
	register("update", "/{project}/{documentType}/{id}", http.MethodPut, canUpdate, update(store, bodyExtractor, withLinks, urls))
	register("delete", "/{project}/{documentType}/{id}", http.MethodDelete, canDelete, delete(store))

	return router
}

func createHandler(
	permit Permit,
	handler func(w Response, r Request),
	allowedDocumentTypes []string,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		project, ok := verifyProject(w, r)
		if !ok {
			return
		}
		documentType, ok := verifyDocumentType(w, r)
		if !ok {
			return
		}

		id, _ := mux.Vars(r)["id"]

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

	return string(t), true
}

// verifyDocumentType checks, that a parameter with the name
// 'documentType' exists. Otherwise, it send an error to the client
func verifyDocumentType(w http.ResponseWriter, r *http.Request) (string, bool) {
	t, exist := mux.Vars(r)["documentType"]

	if !exist || t == "" {
		sendError(w, errors.New("documentType parameter missing"), http.StatusBadRequest)
		return "", false
	}

	return string(t), true
}

func sendError(w http.ResponseWriter, err error, status int) bool {
	w.WriteHeader(status)
	if status >= 500 {
		// Do not let internal messages to the user
		fmt.Fprintln(w, "Internal Server Error")
	} else {
		fmt.Fprintln(w, err)
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
