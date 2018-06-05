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

func Test_Get_Success(t *testing.T) {
	store, _ := jstore.NewStore(memory.DriverName, "")
	router := mux.NewRouter()
	Expose(
		router,
		store,
		allPermited,
		allPermited,
		allPermited,
		allPermited,
		nullQueryExtractor,
		nullBodyExtractor,
		func() interface{} {
			return &TestEntity{}
		},
		func(entity interface{}, links Links) interface{} {
			return TestEntityWithLinks{*entity.(*TestEntity), links}
		},
		documentTypes,
		map[string]string{},
	)
	_, err := store.Marshal(TestEntity{Message: "hello world"}, jstore.NewID("project", "entity", "id"))
	require.NoError(t, err)

	response := getRequest(router, "http://test/project/entity/id")

	require.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "application/json", response.Header().Get("Content-Type"))

	assert.JSONEq(t,
		`{
	"message": "hello world",
	"links": {"self":{"href":"/project/entity/id"}}
}`,
		response.Body.String(),
	)
}

func Test_Get_Failure(t *testing.T) {
	store, _ := jstore.NewStore(memory.DriverName, "")
	router := mux.NewRouter()
	Expose(
		router,
		store,
		allPermited,
		allPermited,
		allPermited,
		allPermited,
		nullQueryExtractor,
		nullBodyExtractor,
		func() interface{} {
			return TestEntity{}
		},
		nullWithLinks,
		documentTypes,
		map[string]string{},
	)

	response := getRequest(router, "http://test/project/entity/not-found")

	require.Equal(t, http.StatusNotFound, response.Code)
}

func Test_Get_ChecksPermits(t *testing.T) {
	store, _ := jstore.NewStore(memory.DriverName, "")
	router := mux.NewRouter()

	Expose(
		router,
		store,
		allPermited,
		nobodyPermited,
		allPermited,
		allPermited,
		nullQueryExtractor,
		nullBodyExtractor,
		nullEntity,
		nullWithLinks,
		documentTypes,
		map[string]string{},
	)

	response := getRequest(router, "http://test/project/entity/id")

	require.Equal(t, http.StatusForbidden, response.Code)
}
