package http

import (
	"net/http"
	"testing"

	"github.com/gorilla/mux"
	jstore "github.com/snabble/go-jstore"
	"github.com/snabble/go-jstore/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Delete_Success(t *testing.T) {
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
			return "id", TestEntity{Message: "hello saturn"}, nil
		},
		func() interface{} {
			return TestEntity{}
		},
		nullWithLinks,
		documentTypes,
	)
	err := store.Marshal(TestEntity{Message: "hello world"}, jstore.ID("project", "entity", "id"))
	require.NoError(t, err)

	response := deleteRequest(router, "http://test/project/entity/id")

	require.Equal(t, http.StatusOK, response.Code)

	stored := TestEntity{}
	err = store.Unmarshal(&stored, "project", "entity", jstore.Id("id"))
	assert.Equal(t, jstore.NotFound, err)
}

func Test_Delete_ChecksPermits(t *testing.T) {
	store, _ := jstore.NewStore(memory.DriverName, "")
	router := mux.NewRouter()

	Expose(
		router,
		store,
		allPermited,
		allPermited,
		allPermited,
		nobodyPermited,
		nullExtractor,
		nullEntity,
		nullWithLinks,
		documentTypes,
	)

	response := deleteRequest(router, "http://test/project/entity/id")

	require.Equal(t, http.StatusForbidden, response.Code)
}
