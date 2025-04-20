package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	objectstorage "github.com/singhmeghna79/homework-object-storage/pkg/internals/objectStorage"
)

// NewPutObjectHandler creates a handler for the PUT /object/{id} endpoint
func HandlePutObject(storageService objectstorage.ObjectStorage) gin.HandlerFunc {
	return func(c *gin.Context) {
		objectID := c.Param("id")

		// Get the size of the request body
		contentLength := c.Request.ContentLength
		if contentLength <= 0 {
			c.JSON(http.StatusBadRequest, BuildResponse("error", "Content-Length header is required", nil))
			return
		}

		// Store the object
		err := storageService.PutObject(c, objectID, c.Request.Body, contentLength)
		if err != nil {
			c.Error(err)

			if fmt.Sprintf("%v", err) == "object ID must contain only alphanumeric characters" ||
				fmt.Sprintf("%v", err) == "object ID must be between 1 and 32 characters" {
				c.JSON(http.StatusBadRequest, BuildResponse("error", err.Error(), nil))
				return
			}

			c.JSON(http.StatusInternalServerError, BuildResponse("error", "Failed to store object", nil))
			return
		}

		c.JSON(http.StatusCreated, BuildResponse("success", fmt.Sprintf("Object %s stored successfully", objectID), nil))
	}
}
