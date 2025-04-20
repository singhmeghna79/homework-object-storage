package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/singhmeghna79/homework-object-storage/pkg/internals/objectStorage/fakes"
	"github.com/stretchr/testify/assert"
)

func TestHandlePutbject(t *testing.T) {
	// Set Gin to Test Mode
	gin.SetMode(gin.TestMode)

	objectStorageFailure1 := &fakes.InterfaceObjectStorage{}
	objectStorageFailure1.PutObjectReturns(fmt.Errorf("object ID must contain only alphanumeric characters"))

	objectStorageFailure2 := &fakes.InterfaceObjectStorage{}
	objectStorageFailure2.PutObjectReturns(errors.New("object ID must be between 1 and 32 characters"))

	objectStorageFailure3 := &fakes.InterfaceObjectStorage{}
	objectStorageFailure3.PutObjectReturns(errors.New("Failed to store object"))

	objectStorageSuccess := &fakes.InterfaceObjectStorage{}
	objectStorageSuccess.PutObjectReturns(nil)

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
			expectedStatus:    http.StatusCreated,
			expectedResponse:  `{"message":"Object testobject stored successfully","status":"success"}`,
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
			expectedStatus:    http.StatusInternalServerError,
			expectedResponse:  `{"status":"error","message":"Failed to store object"}`,
			checkBody:         true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a new Gin router
			router := gin.New()

			// Make sure we're using the correct fake for each test case
			router.PUT("/object/:id", HandlePutObject(tc.objectStorageFake))

			// Create a test request
			req, _ := http.NewRequest(http.MethodPut, "/object/"+tc.objectID, nil)

			// Set the ContentLength field directly
			req.ContentLength = int64(123)
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
