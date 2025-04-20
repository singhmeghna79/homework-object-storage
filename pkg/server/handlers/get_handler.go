package handlers

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	objectstorage "github.com/singhmeghna79/homework-object-storage/pkg/internals/objectStorage"
)

func HandleGetObject(storageService objectstorage.ObjectStorage) gin.HandlerFunc {
	return func(c *gin.Context) {
		objectID := c.Param("id")

		// Retrieve the object
		obj, err := storageService.GetObject(c, objectID)
		if err != nil {
			c.Error(err)

			if fmt.Sprintf("%v", err) == "object ID must contain only alphanumeric characters" ||
				fmt.Sprintf("%v", err) == "object ID must be between 1 and 32 characters" {
				c.JSON(http.StatusBadRequest, BuildResponse("error", err.Error(), nil))
				return
			}

			if fmt.Sprintf("%v", err) == "object not found" {
				c.JSON(http.StatusNotFound, BuildResponse("error", "Object not found", nil))
				return
			}

			c.JSON(http.StatusInternalServerError, BuildResponse("error", "Failed to retrieve object", nil))
			return
		}
		defer obj.Close()

		// Set content type based on object metadata
		c.Writer.Header().Set("Content-Type", "application/octet-stream")

		// Copy the object to the response
		_, err = io.Copy(c.Writer, obj)
		if err != nil {
			c.Error(err)
			c.JSON(http.StatusInternalServerError, BuildResponse("error", "Failed to send object", nil))
			return
		}
	}
}
