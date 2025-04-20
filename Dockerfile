FROM golang:1.23-alpine

# Set working directory
WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy all files into the container
ADD . .

# Display contents for debugging (optional)
RUN ls -la

# Download Go module dependencies
RUN go mod tidy

# Build the binary
RUN go build -v -o minio-storage-plugin ./cmd/main.go

# Confirm the binary is built and is executable
RUN ls -la minio-storage-plugin && chmod +x minio-storage-plugin

# Set the entrypoint
CMD ["./minio-storage-plugin"]