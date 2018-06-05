package http

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	jstore "github.com/snabble/go-jstore"
	"github.com/snabble/go-jstore/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Integration(t *testing.T) {
	store, _ := jstore.NewStore(memory.DriverName, "")
	router := mux.NewRouter()
	Expose(
		router,
		store,
		allPermited,
		allPermited,
		allPermited,
		allPermited,
		func(r Request) (limit int, query []jstore.Option, err error) {
			queries := r.OriginalRequest.URL.Query()
			limitStr, ok := queries["limit"]
			limit = 1000
			if ok {
				limit, _ = strconv.Atoi(limitStr[0])
			}

			propertyQuery, ok := queries["property"]
			if ok {
				query = []jstore.Option{jstore.Eq("property", propertyQuery[0])}
			}

			return limit, query, nil
		},
		func(r Request) (string, interface{}, error) {
			id := r.ID
			if id == "" {
				generatedID, _ := uuid.NewRandom()
				id = generatedID.String()
			}

			entity := TestEntity{}
			err := r.UnmarshalBody(&entity)
			return id, entity, err
		},
		func() interface{} {
			return &TestEntity{}
		},
		nullWithLinks,
		documentTypes,
		map[string]string{"entity": "entities"},
	)

	var location string
	t.Run("create", func(t *testing.T) {
		body := `{"message":"hello world", "property": "nice"}`

		response := postRequest(router, "http://test/project/entities", body)

		require.Equal(t, http.StatusCreated, response.Code)
		assert.JSONEq(t, `{ "message": "hello world", "property": "nice" }`, response.Body.String())

		location = response.Header().Get("Location")

	})

	t.Run("get", func(t *testing.T) {
		response := getRequest(router, location)

		require.Equal(t, http.StatusOK, response.Code)
		assert.JSONEq(t, `{ "message": "hello world", "property": "nice" }`, response.Body.String())
	})

	// create more entities
	postRequest(router, "http://test/project/entities", `{"message":"hello mars", "property": "ok"}`)
	postRequest(router, "http://test/project/entities", `{"message":"hello saturn", "property": "notok"}`)

	t.Run("list with limit", func(t *testing.T) {
		response := getRequest(router, "http://test/project/entities?limit=1")

		require.Equal(t, http.StatusOK, response.Code)
	})

	t.Run("list with query", func(t *testing.T) {
		response := getRequest(router, "http://test/project/entities?property=nice")

		require.Equal(t, http.StatusOK, response.Code)
		assert.JSONEq(t, `{
	"resources": [{ "message": "hello world", "property": "nice" }],
	"links": {"self": {"href": "/project/entities"}}
}`,
			response.Body.String(),
		)
	})

	t.Run("update", func(t *testing.T) {
		response := putRequest(router, location, `{ "message": "hello jstore" }`)

		require.Equal(t, http.StatusOK, response.Code)
		assert.JSONEq(t, `{ "message": "hello jstore" }`, response.Body.String())

		t.Run("is reflected in the resource", func(t *testing.T) {
			response := getRequest(router, location)

			require.Equal(t, http.StatusOK, response.Code)
			assert.JSONEq(t, `{ "message": "hello jstore" }`, response.Body.String())
		})
	})

	t.Run("delete", func(t *testing.T) {
		response := deleteRequest(router, location)

		require.Equal(t, http.StatusOK, response.Code)

		t.Run("is reflected in the resource", func(t *testing.T) {
			response := getRequest(router, location)

			require.Equal(t, http.StatusNotFound, response.Code)
		})
	})
}
