package handlers

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/singhmeghna79/homework-object-storage/pkg/internals/objectStorage/fakes"
	"github.com/stretchr/testify/assert"
)

type mockReadCloser struct {
	*bytes.Reader
}

func (m mockReadCloser) Close() error {
	return nil
}

func newMockReadCloser(data string) io.ReadCloser {
	return mockReadCloser{bytes.NewReader([]byte(data))}
}

func TestHandleGetObject(t *testing.T) {
	// Set Gin to Test Mode
	gin.SetMode(gin.TestMode)

	objectStorageFailure1 := &fakes.InterfaceObjectStorage{}
	objectStorageFailure1.GetObjectReturns(nil, fmt.Errorf("object ID must contain only alphanumeric characters"))

	objectStorageFailure2 := &fakes.InterfaceObjectStorage{}
	objectStorageFailure2.GetObjectReturns(nil, errors.New("object ID must be between 1 and 32 characters"))

	objectStorageFailure3 := &fakes.InterfaceObjectStorage{}
	objectStorageFailure3.GetObjectReturns(nil, errors.New("object not found"))

	objectStorageSuccess := &fakes.InterfaceObjectStorage{}
	objectStorageSuccess.GetObjectReturns(newMockReadCloser("test content"), nil)

	// Test cases
	testCases := []struct {
		name              string
		objectID          string
		objectStorageFake *fakes.InterfaceObjectStorage
		expectedStatus    int
		expectedResponse  string
		checkBody         bool
	}{
		{
			name:              "Success",
			objectID:          "testobject",
			objectStorageFake: objectStorageSuccess,
			expectedStatus:    http.StatusOK,
			expectedResponse:  "test content",
			checkBody:         true,
		},
		{
			name:              "Invalid ObjectID Characters",
			objectID:          "test-object!",
			objectStorageFake: objectStorageFailure1,
			expectedStatus:    http.StatusBadRequest,
			expectedResponse:  `{"status":"error","message":"object ID must contain only alphanumeric characters"}`,
			checkBody:         true,
		},
		{
			name:              "Invalid ObjectID Length",
			objectID:          "testobjectthatiswaytoolongforthelimitsofthesystem",
			objectStorageFake: objectStorageFailure2,
			expectedStatus:    http.StatusBadRequest,
			expectedResponse:  `{"status":"error","message":"object ID must be between 1 and 32 characters"}`,
			checkBody:         true,
		},
		{
			name:              "Object Not Found",
			objectID:          "nonexistent",
			objectStorageFake: objectStorageFailure3,
			expectedStatus:    http.StatusNotFound,
			expectedResponse:  `{"status":"error","message":"Object not found"}`,
			checkBody:         true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a new Gin router
			router := gin.New()

			// Make sure we're using the correct fake for each test case
			router.GET("/object/:id", HandleGetObject(tc.objectStorageFake))

			// Create a test request
			req, _ := http.NewRequest(http.MethodGet, "/object/"+tc.objectID, nil)
			resp := httptest.NewRecorder()

			// Serve the request
			router.ServeHTTP(resp, req)

			// Check status code
			assert.Equal(t, tc.expectedStatus, resp.Code)

			// Check response body if needed
			if tc.checkBody {
				if tc.expectedStatus == http.StatusOK {
					// For success case, we expect the raw object content
					assert.Equal(t, tc.expectedResponse, resp.Body.String())
				} else {
					// For error cases, we expect a JSON response
					assert.JSONEq(t, tc.expectedResponse, resp.Body.String())
				}
			}
		})
	}
}
