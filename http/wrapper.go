package http

import (
	"encoding/json"
	"net/http"

	jstore "github.com/snabble/go-jstore"
)

// Request to store
type Request struct {
	// The original http request
	OriginalRequest *http.Request

	// The requested Project id
	Project string
	// The requested DocumentType
	DocumentType string
	// Id of the requested object
	ID string
}

func (request *Request) UnmarshalBody(obj interface{}) error {
	decoder := json.NewDecoder(request.OriginalRequest.Body)
	err := decoder.Decode(obj)
	if err != nil {
		return WrapWithClientError(err)
	}
	return nil
}

// Response wrapper
type Response struct {
	Writer http.ResponseWriter
}

// SendError sends the appropriate status code to the client
func (response *Response) SendError(err error) {
	sendError(response.Writer, err, selectStatusCode(err))
}

// AddHeader to the response
func (response *Response) AddHeader(name, value string) {
	response.Writer.Header().Add(name, value)
}

// Send the status code and serialized object
func (response *Response) Send(statusCode int, obj interface{}) {
	out, err := json.Marshal(obj)
	if err != nil {
		sendError(response.Writer, err, http.StatusInternalServerError)
		return
	}
	response.Writer.Header().Set("Content-Type", "application/json")
	response.Writer.WriteHeader(statusCode)
	response.Writer.Write(out)
}

func selectStatusCode(err error) int {
	switch err.(type) {
	case Error:
		if err.(Error).IsClientError() {
			return http.StatusBadRequest
		}
		return http.StatusInternalServerError
	default:
		if err == jstore.NotFound {
			return http.StatusNotFound
		}
	}
	return http.StatusInternalServerError
}
