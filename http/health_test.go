package http

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func Test_Health(t *testing.T) {
	for _, test := range []struct {
		name string

		permitted     bool
		returnedError error

		expectedResponseCode int
		expectedContentType  string
	}{
		{
			name: "status ok",

			permitted:     true,
			returnedError: nil,

			expectedResponseCode: http.StatusOK,
			expectedContentType:  "application/json",
		},
		{
			name: "status down",

			permitted:     true,
			returnedError: errors.New("an error"),

			expectedResponseCode: http.StatusInternalServerError,
			expectedContentType:  "application/json",
		},
		{
			name: "request not permited",

			permitted: false,

			expectedResponseCode: http.StatusForbidden,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			router := mux.NewRouter()
			HealthRoute(
				router,
				func() error {
					return test.returnedError
				},
				func(r *http.Request) bool {
					return test.permitted
				},
			)
			r := &http.Request{
				Method: http.MethodGet,
				URL:    &url.URL{Path: "/health"},
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, r)

			assert.Equal(t, test.expectedResponseCode, w.Code)
			if test.expectedContentType != "" {
				assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
			}
		})
	}
}
