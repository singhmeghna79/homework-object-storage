# Distributed Object Storage Gateway

A stateless distributed Object Storage Gateway that provides a unified API interface to multiple MinIO instances. This gateway automatically discovers MinIO nodes running in Docker and distributes objects across them using consistent hashing.

## Architecture Overview

The gateway acts as a single entry point for accessing objects stored across multiple MinIO instances. Key features include:

- Dynamic discovery of MinIO nodes through Docker API
- Automatic client initialization for discovered nodes
- Consistent object distribution using FNV-1a hash
- Automatic bucket creation on startup
- Structured logging with Zap
- Request tracing with unique IDs
- Proper error handling and status responses

## Prerequisites

- Docker and Docker Compose installed
- Go 1.21+ (for development only)

## Getting Started

1. Clone the repository:
```bash
git clone https://github.com/singhmeghna79/homework-object-storage
cd homework-object-storage
```

2. Start the services using Docker Compose:
```bash
docker-compose up --build
```

This will start:
- 3 MinIO nodes (ports 9001-9003 for console access)
- Gateway service (port 3000)

## API Endpoints

### Health Check
```bash
GET /health
```
Check if the service is running properly.

Example:
```bash
curl http://localhost:3000/health
```

### Store Object
```bash
PUT /api/v1/object/{id}
```
Upload an object with the specified ID.

Example:
```bash
curl -X PUT -d "This is a test object" http://localhost:3000/api/v1/object/test123
```

### Retrieve Object
```bash
GET /api/v1/object/{id}
```
Download an object by ID.

Example:
```bash
curl http://localhost:3000/api/v1/object/test123
```

## Object ID Requirements

Object IDs must:
- Be alphanumeric characters only 
- Have length between 1 and 32 characters

## Key Components

### Main Gateway Service
- Runs on port 3000
- Discovers MinIO nodes automatically through Docker
- Routes requests based on object ID hashing

### MinIO Nodes
- 3 pre-configured MinIO instances
- Each has unique credentials (automatically configured)
- Accessible via web console at ports 9001-9003

### Docker Network
- Custom network (169.253.0.0/24) for inter-service communication
- Gateway has access to Docker socket for node discovery

## Development

### Project Structure
```
├── cmd/main.go                   # Entry point
├── pkg/
│   ├── server/              # HTTP server implementation
│   ├── internals/
│   │   ├── dockerClient/    # Docker client for node discovery
│   │   ├── objectStorage/   # Storage interface and MinIO implementation
│   │   └── handlers/        # HTTP request handlers
```

### Configuration

Command line flags:
- `--port`: HTTP server port (default: 3000)
- `--storageType`: Object storage type (default: minio)

Environment variables and credentials are auto-discovered through Docker.

## Monitoring

### Logs
All components use structured JSON logging with Zap. Each request includes:
- Unique request ID
- Processing latency
- Status code
- Path and method

### Health Monitoring
The `/health` endpoint can be used for liveness probes.

### MinIO Consoles
Access individual MinIO instances:
- Node 1: http://localhost:9001
- Node 2: http://localhost:9002
- Node 3: http://localhost:9003

## Design Decisions

### Object Distribution
Objects are distributed using FNV-1a hash function on the object ID, ensuring:
- Consistent mapping of IDs to nodes
- Even distribution across nodes
- Fast lookup times

### Stateless Design
The gateway is completely stateless:
- No local storage or caching
- Node discovery happens at startup
- All state managed by MinIO nodes

### Error Handling
The service implements comprehensive error handling:
- Invalid object IDs (400 Bad Request)
- Non-existent objects (404 Not Found)
- Service failures (500 Internal Server Error)

## Testing

The service was tested with:
```bash
# Check service health
curl http://localhost:3000/health
{"message":"OK","status":"health"}

# Try to get non-existent object
curl http://localhost:3000/api/v1/object/test123
{"message":"Object not found","status":"error"}

# Upload an object
curl -X PUT -d "This is a test object" http://localhost:3000/api/v1/object/test123
{"message":"Object test123 stored successfully","status":"success"}

# Retrieve the object
curl http://localhost:3000/api/v1/object/test123
This is a test object
```

## Future Enhancements

Potential improvements:
- Add support for object metadata
- Implement list operations
- Add health checks for MinIO nodes
- Support for object versioning (currently overriding the if request comes for same object)
- Authentication and authorization
- Adding persistent volume and db, so that restarts of minio nodes and gateway server doesnot lose that data

