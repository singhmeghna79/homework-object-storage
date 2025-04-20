package objectStorage

import (
	"context"
	"io"

	"github.com/gin-gonic/gin"
	docker "github.com/singhmeghna79/homework-object-storage/pkg/internals/dockerClient"
	"go.uber.org/zap"
)

const (
	minioStorage = "minio"
)

//go:generate counterfeiter -o fakes/InterfaceObjectStorage.go --fake-name InterfaceObjectStorage . ObjectStorage
type ObjectStorage interface {
	GetObject(ctx *gin.Context, objectID string) (io.ReadCloser, error)
	PutObject(ctx *gin.Context, objectID string, data io.Reader, size int64) error
	// ListObject(nodeId string, bucketName string)
}

type objectStorageFactory struct {
}

// StorageGeneric shall be a generic implementation of the object storage interface
// There can be some object storage which shall use this common method for storing and retrieval
// of objects
type StorageGeneric struct {
}

func newStorageGeneric(logger *zap.Logger) ObjectStorage {
	// log := internals.GetLogger(c)
	dockerClient, err := docker.NewClient()
	if err != nil {
		logger.Fatal("Failed to create Docker client: %v", zap.Error(err))
	}

	// Discover Minio nodes
	minioNodes, err := dockerClient.DiscoverMinioNodes(context.Background())
	if err != nil {
		logger.Fatal("Failed to discover Minio nodes: %v", zap.Error(err))
	}

	if len(minioNodes) == 0 {
		logger.Fatal("No Minio nodes found")
	}

	logger.Info("Discovered Minio nodes", zap.Int("minio nodes", len(minioNodes)))

	// Create minio service
	minioStorageService := NewminioStorageService(minioNodes, logger)
	return minioStorageService
}

func (p *objectStorageFactory) GetObjectStorage(objectStorageType string, logger *zap.Logger) ObjectStorage {
	switch {
	case (objectStorageType == minioStorage):
		return newStorageGeneric(logger)
	default:
		return newStorageGeneric(logger)
	}
}

func NewObjectStorageFactory() *objectStorageFactory {
	return &objectStorageFactory{}
}
