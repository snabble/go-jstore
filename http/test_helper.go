package http

import (
	"bytes"
	"net/http"
	"net/http/httptest"

	jstore "github.com/snabble/go-jstore/v2"
)

type TestEntity struct {
	Message  string `json:"message"`
	Property string `json:"property,omitempty"`
}

type TestEntityWithLinks struct {
	TestEntity
	Links Links `json:"links"`
}

var (
	documentTypes     = []string{"entity"}
	allPermited       = func(r Request) bool { return true }
	nobodyPermited    = func(r Request) bool { return false }
	nullBodyExtractor = func(r Request) (string, interface{}, error) {
		return "id", TestEntity{Message: ""}, nil
	}
	nullQueryExtractor = func(r Request) (limit int, query []jstore.Option, err error) {
		return 100, []jstore.Option{}, nil
	}
	nullEntity = func() interface{} {
		return TestEntity{}
	}
	nullWithLinks = func(entity interface{}, links Links) interface{} {
		return entity
	}
)

func getRequest(h http.Handler, url string) *httptest.ResponseRecorder {
	return executeRequest(h, url, http.MethodGet, "")
}

func postRequest(h http.Handler, url string, body string) *httptest.ResponseRecorder {
	return executeRequest(h, url, http.MethodPost, body)
}

func putRequest(h http.Handler, url string, body string) *httptest.ResponseRecorder {
	return executeRequest(h, url, http.MethodPut, body)
}

func deleteRequest(h http.Handler, url string) *httptest.ResponseRecorder {
	return executeRequest(h, url, http.MethodDelete, "")
}

func executeRequest(h http.Handler, url, method string, body string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, url, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)
	return resp
}
