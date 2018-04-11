package http

import (
	"errors"
	"net/http"
	"testing"

	"github.com/gorilla/mux"
	jstore "github.com/snabble/go-jstore"
	"github.com/snabble/go-jstore/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Update_Success(t *testing.T) {
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
	err := store.Marshal(TestEntity{Message: "hello world"}, "project", "entity", "id")
	require.NoError(t, err)

	response := putRequest(router, "http://test/project/entity/id", `{"message":"hello saturn"}`)

	require.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "application/json", response.Header().Get("Content-Type"))

	assert.JSONEq(t, `{ "message": "hello saturn" }`, response.Body.String())

	stored := TestEntity{}
	err = store.Unmarshal(&stored, "project", "entity", jstore.Id("id"))
	assert.NoError(t, err)
	assert.Equal(t, TestEntity{Message: "hello saturn"}, stored)
}

func Test_Update_NotFound(t *testing.T) {
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
			return r.ID, TestEntity{Message: "hello"}, nil
		},
		func() interface{} {
			return TestEntity{}
		},
		nullWithLinks,
		documentTypes,
	)

	response := putRequest(router, "http://test/project/entity/not-present", `{"message": "hello"}`)

	require.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `{ "message": "hello" }`, response.Body.String())

	stored := TestEntity{}
	err := store.Unmarshal(&stored, "project", "entity", jstore.Id("not-present"))
	assert.NoError(t, err)
	assert.Equal(t, TestEntity{Message: "hello"}, stored)
}

func Test_Update_ExtractionErrors(t *testing.T) {
	for _, test := range []struct {
		name           string
		returnedError  error
		expectedStatus int
	}{
		{
			name:           "some error",
			returnedError:  errors.New("some error"),
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "internal error",
			returnedError:  InternalError("something"),
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "client error",
			returnedError:  ClientError("something"),
			expectedStatus: http.StatusBadRequest,
		},
	} {
		t.Run(test.name, func(t *testing.T) {

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
					return "", nil, test.returnedError
				},
				func() interface{} { return TestEntity{} },
				nullWithLinks,
				documentTypes,
			)
			body := `{"message":"hello world"}`

			response := putRequest(router, "http://test/project/entity/id", body)

			require.Equal(t, test.expectedStatus, response.Code)
		})
	}
}

func Test_Update_ValidatesProvidedId(t *testing.T) {
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
			return "id", TestEntity{Message: "hello"}, nil
		},
		func() interface{} {
			return TestEntity{}
		},
		nullWithLinks,
		documentTypes,
	)

	response := putRequest(router, "http://test/project/entity/another-id", `{"message": "hello"}`)

	require.Equal(t, http.StatusBadRequest, response.Code)
}

func Test_Update_ChecksPermits(t *testing.T) {
	store, _ := jstore.NewStore(memory.DriverName, "")
	router := mux.NewRouter()

	Expose(
		router,
		store,
		allPermited,
		allPermited,
		nobodyPermited,
		allPermited,
		nullExtractor,
		nullEntity,
		nullWithLinks,
		documentTypes,
	)

	response := putRequest(router, "http://test/project/entity/id", "")

	require.Equal(t, http.StatusForbidden, response.Code)
}
