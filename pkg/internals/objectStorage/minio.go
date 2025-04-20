package objectStorage

import (
	"context"
	"fmt"
	"hash/fnv"
	"io"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	utils "github.com/singhmeghna79/homework-object-storage/pkg"
	docker "github.com/singhmeghna79/homework-object-storage/pkg/internals/dockerClient"
	"go.uber.org/zap"
)

const bucketName = "objects"

// Service handles object storage operations
type minioStorageService struct {
	nodes        []docker.MinioNode
	clients      map[string]*minio.Client
	clientsMutex sync.RWMutex
	logger       *zap.Logger
}

// NewService creates a new gateway service
func NewminioStorageService(nodes []docker.MinioNode, logger *zap.Logger) *minioStorageService {
	service := &minioStorageService{
		nodes:   nodes,
		clients: make(map[string]*minio.Client),
		logger:  logger,
	}

	// Initialize Minio clients for each node
	for _, node := range nodes {

		for i := 0; i < 5; i++ {
			err := service.initializeClient(node)
			if err == nil {
				break
			}
			logger.Info("Retrying MinIO client init ...", zap.Int("retry", i+1))
			time.Sleep(10 * time.Second)
		}
		if err := service.initializeClient(node); err != nil {
			logger.Error("Failed to initialize client for node ", zap.String("node_name", node.Name), zap.Error(err))
		}
	}

	return service
}

// initializeClient creates a Minio client for a given node
func (s *minioStorageService) initializeClient(node docker.MinioNode) error {
	endpoint := fmt.Sprintf("%s:%s", node.IPAddress, node.Port)
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(node.AccessKey, node.SecretKey, ""),
		Secure: false, // Using HTTP, not HTTPS
	})
	if err != nil {
		return fmt.Errorf("failed to create Minio client: %w", err)
	}

	// Create the bucket if it doesn't exist
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exists, err := client.BucketExists(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("failed to check if bucket exists: %w", err)
	}

	if !exists {
		err = client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
		s.logger.Info("Created bucket on node ", zap.String("bucketName", bucketName), zap.String("node_name", node.Name))
	}

	s.clientsMutex.Lock()
	s.clients[node.ID] = client
	s.clientsMutex.Unlock()

	s.logger.Info("Initialized client for node at ", zap.String("node_name", node.Name), zap.String("endpoint", endpoint))
	return nil
}

// getNodeForID consistently maps an object ID to a specific node
func (s *minioStorageService) getNodeForID(objectID string) (docker.MinioNode, *minio.Client, error) {
	if len(s.nodes) == 0 {
		return docker.MinioNode{}, nil, fmt.Errorf("no storage nodes available")
	}

	// Use a simple hash function to consistently map the object ID to a node
	h := fnv.New32a()
	h.Write([]byte(objectID))
	nodeIndex := int(h.Sum32()) % len(s.nodes)
	node := s.nodes[nodeIndex]

	s.clientsMutex.RLock()
	client, exists := s.clients[node.ID]
	s.clientsMutex.RUnlock()

	if !exists {
		return node, nil, fmt.Errorf("client for node %s not initialized", node.Name)
	}

	return node, client, nil
}

// PutObject stores an object in the appropriate node
func (s *minioStorageService) PutObject(ctx *gin.Context, objectID string, data io.Reader, size int64) error {
	logger := utils.GetLogger(ctx)
	// Validate object ID
	if err := validateObjectID(objectID); err != nil {
		return err
	}

	// Get the appropriate node and client
	node, client, err := s.getNodeForID(objectID)
	if err != nil {
		return err
	}

	logger.Info("Storing object on node ", zap.String("object_id", objectID), zap.String("node_name", node.Name))

	// Upload the object
	_, err = client.PutObject(ctx, bucketName, objectID, data, size, minio.PutObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to store object: %w", err)
	}

	return nil
}

// GetObject retrieves an object from the appropriate node
func (s *minioStorageService) GetObject(ctx *gin.Context, objectID string) (io.ReadCloser, error) {
	logger := utils.GetLogger(ctx)
	// Validate object ID
	if err := validateObjectID(objectID); err != nil {
		return nil, err
	}

	// Get the appropriate node and client
	node, client, err := s.getNodeForID(objectID)
	if err != nil {
		return nil, err
	}

	logger.Info("Retrieving object from node ", zap.String("object_id", objectID), zap.String("node_name", node.Name))

	// Get the object
	obj, err := client.GetObject(ctx, bucketName, objectID, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}

	// Check if the object exists by attempting to get its stats
	_, err = obj.Stat()
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return nil, fmt.Errorf("object not found")
		}
		return nil, fmt.Errorf("failed to stat object: %w", err)
	}

	return obj, nil
}

// validateObjectID ensures the object ID meets the requirements
func validateObjectID(id string) error {
	if len(id) == 0 || len(id) > 32 {
		return fmt.Errorf("object ID must be between 1 and 32 characters")
	}

	for _, char := range id {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9')) {
			return fmt.Errorf("object ID must contain only alphanumeric characters")
		}
	}

	return nil
}
