// main.go
package main

import (
	"flag"
	"os"

	"github.com/singhmeghna79/homework-object-storage/pkg/server"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	// Parse command line flags
	serverPort := flag.String("port", "3000", "HTTP server port")
	storageType := flag.String("storageType", "minio", "Object Storage Type")
	flag.Parse()

	// Setup logger
	logger := setUpLogger()
	defer logger.Sync()

	// Initialize and run server
	srv := server.New(*serverPort, *storageType, logger)
	srv.Run()

	os.Exit(0)
}

func setUpLogger() *zap.Logger {
	// Prepare a new logger
	atom := zap.NewAtomicLevel()
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "timestamp"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	logger := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.Lock(os.Stdout),
		atom,
	), zap.AddCaller()).With(zap.String("service", "amazin-api"))
	atom.SetLevel(zap.InfoLevel)
	return logger
}
