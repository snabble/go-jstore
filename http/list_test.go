package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/gorilla/mux"
	jstore "github.com/snabble/go-jstore"
	"github.com/snabble/go-jstore/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_List_Success(t *testing.T) {
	store, _ := jstore.NewStore(memory.DriverName, "")
	router := mux.NewRouter()
	Expose(
		router,
		store,
		allPermited,
		allPermited,
		allPermited,
		allPermited,
		nullExtractor,
		func() interface{} {
			return &TestEntity{}
		},
		func(entity interface{}, links Links) interface{} {
			return TestEntityWithLinks{*entity.(*TestEntity), links}
		},
		documentTypes,
	)
	store.Marshal(TestEntity{Message: "hello world"}, jstore.NewID("project", "entity", "earth"))
	store.Marshal(TestEntity{Message: "hello saturn"}, jstore.NewID("project", "entity", "saturn"))
	store.Marshal(TestEntity{Message: "hello mars"}, jstore.NewID("project", "entity", "mars"))

	response := getRequest(router, "http://test/project/entity")

	require.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "application/json", response.Header().Get("Content-Type"))

	assertEntitiesListJSONEqual(t,
		`{
	"resources": [
		{
			"message": "hello world",
			"links": {"self": {"href": "/project/entity/earth"}}
		},
		{
			"message": "hello saturn",
			"href": "/project/entity/saturn"
		},
		{
			"message": "hello mars",
			"href": "/project/entity/mars"
		}
	],
	"links": {"self": {"href": "/project/entity"}}
}`,
		response.Body.String(),
	)
}

func Test_List_Empty(t *testing.T) {
	store, _ := jstore.NewStore(memory.DriverName, "")
	router := mux.NewRouter()
	Expose(
		router,
		store,
		allPermited,
		allPermited,
		allPermited,
		allPermited,
		nullExtractor,
		func() interface{} {
			return TestEntity{}
		},
		nullWithLinks,
		documentTypes,
	)

	response := getRequest(router, "http://test/not-exists/entity")

	require.Equal(t, http.StatusNotFound, response.Code)
}

func Test_List_ChecksPermits(t *testing.T) {
	store, _ := jstore.NewStore(memory.DriverName, "")
	router := mux.NewRouter()

	Expose(
		router,
		store,
		allPermited,
		nobodyPermited,
		allPermited,
		allPermited,
		nullExtractor,
		nullEntity,
		nullWithLinks,
		documentTypes,
	)

	response := getRequest(router, "http://test/project/entity")

	require.Equal(t, http.StatusForbidden, response.Code)
}

type TestEntityList struct {
	Resources []TestEntity `json:"resources"`
	Links     Links        `json:"links"`
}

func assertEntitiesListJSONEqual(t *testing.T, expected, actual string) {
	var expectedList, actualList TestEntityList

	if err := json.Unmarshal([]byte(expected), &expectedList); err != nil {
		assert.Fail(t, fmt.Sprintf("Expected value ('%s') is not valid json.\nJSON parsing error: '%s'", expected, err.Error()))
		return
	}

	if err := json.Unmarshal([]byte(actual), &actualList); err != nil {
		assert.Fail(t, fmt.Sprintf("Input ('%s') needs to be valid json.\nJSON parsing error: '%s'", actual, err.Error()))
		return
	}

	assert.Equal(t, expectedList.Links, actualList.Links)
	assert.Subset(t, expectedList.Resources, actualList.Resources)
	assert.Subset(t, actualList.Resources, expectedList.Resources)
}
