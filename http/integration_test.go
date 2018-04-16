package http

import (
	"net/http"
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
	)

	var location string
	t.Run("create", func(t *testing.T) {
		body := `{"message":"hello world"}`

		response := postRequest(router, "http://test/project/entity", body)

		require.Equal(t, http.StatusCreated, response.Code)
		assert.JSONEq(t, `{ "message": "hello world" }`, response.Body.String())

		location = response.Header().Get("Location")
	})

	t.Run("get", func(t *testing.T) {
		response := getRequest(router, location)

		require.Equal(t, http.StatusOK, response.Code)
		assert.JSONEq(t, `{ "message": "hello world" }`, response.Body.String())
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
