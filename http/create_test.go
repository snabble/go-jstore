package http

import (
	"errors"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/gorilla/mux"
	jstore "github.com/snabble/go-jstore/v2"
	"github.com/snabble/go-jstore/v2/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Create_Success(t *testing.T) {
	store, _ := jstore.NewStore(memory.DriverName, "")
	router := mux.NewRouter()
	message := "hello world"

	var requestArg Request
	Expose(
		router,
		store,
		allPermited,
		allPermited,
		allPermited,
		allPermited,
		nullQueryExtractor,
		func(r Request) (string, interface{}, error) {
			requestArg = r
			return "id", TestEntity{Message: message}, nil
		},
		func() interface{} {
			return TestEntity{}
		},
		nullWithLinks,
		documentTypes,
		map[string]string{},
	)
	body := `{"message":"hello world"}`

	response := postRequest(router, "http://test/project/entity", body)

	require.Equal(t, http.StatusCreated, response.Code)
	assert.NotEmpty(t, response.Header().Get("Location"))

	assert.Equal(t, "project", requestArg.Project)
	assert.Equal(t, "entity", requestArg.DocumentType)

	bodyBytes, err := ioutil.ReadAll(response.Body)
	require.NoError(t, err)
	assert.Equal(t, body, string(bodyBytes))
}

func Test_Create_Success_Without_Body_Response(t *testing.T) {
	store, _ := jstore.NewStore(memory.DriverName, "")
	router := mux.NewRouter()
	message := "hello world"

	var requestArg Request
	Expose(
		router,
		store,
		allPermited,
		allPermited,
		allPermited,
		allPermited,
		nullQueryExtractor,
		func(r Request) (string, interface{}, error) {
			requestArg = r
			return "id", TestEntity{Message: message}, nil
		},
		func() interface{} {
			return TestEntity{}
		},
		nullWithLinks,
		documentTypes,
		map[string]string{},
		PostDoNotRespondWithBody(),
	)
	body := `{"message":"hello world"}`

	response := postRequest(router, "http://test/project/entity", body)

	require.Equal(t, http.StatusCreated, response.Code)
	assert.NotEmpty(t, response.Header().Get("Location"))

	assert.Equal(t, "entity", requestArg.DocumentType)

	bodyBytes, err := ioutil.ReadAll(response.Body)
	require.NoError(t, err)
	assert.Equal(t, "", string(bodyBytes))
}

func Test_Create_ExtractionErrors(t *testing.T) {
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
				nullQueryExtractor,
				func(r Request) (string, interface{}, error) {
					return "", nil, test.returnedError
				},
				func() interface{} { return TestEntity{} },
				nullWithLinks,
				documentTypes,
				map[string]string{},
			)
			body := `{"message":"hello world"}`

			response := postRequest(router, "http://test/project/entity", body)

			require.Equal(t, test.expectedStatus, response.Code)
		})
	}
}

func Test_Create_ChecksPermits(t *testing.T) {
	store, _ := jstore.NewStore(memory.DriverName, "")
	router := mux.NewRouter()

	Expose(
		router,
		store,
		nobodyPermited,
		allPermited,
		allPermited,
		allPermited,
		nullQueryExtractor,
		nullBodyExtractor,
		nullEntity,
		nullWithLinks,
		documentTypes,
		map[string]string{},
	)

	response := postRequest(router, "http://test/project/entity", `{"message":"hello world"}`)

	require.Equal(t, http.StatusForbidden, response.Code)
}
