package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/gofiber/fiber/v2"
	appconfig "github.com/pixelados-net/discord-bot/platform/app"
	"go.uber.org/zap"
)

// Server runs a Fiber application with graceful shutdown.
type Server struct {
	application *fiber.App
	config      appconfig.Config
	log         *zap.Logger
	version     string
}

// NewServer creates an HTTP server.
func NewServer(application *fiber.App, config appconfig.Config, log *zap.Logger, version string) *Server {
	return &Server{application: application, config: config, log: log, version: version}
}

// Run listens until the context is canceled or the server fails.
func (server *Server) Run(ctx context.Context) error {
	listener, err := net.Listen("tcp", server.config.Address())
	if err != nil {
		return fmt.Errorf("listen on %s: %w", server.config.Address(), err)
	}
	server.log.Info("starting discord-bot server",
		zap.String("environment", string(server.config.Environment)),
		zap.String("host", server.config.Host),
		zap.Int("port", server.config.Port),
		zap.String("version", server.version),
	)
	errorsChannel := make(chan error, 1)
	go func() { errorsChannel <- server.application.Listener(listener) }()
	select {
	case <-ctx.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.application.ShutdownWithContext(shutdownContext); err != nil {
			return fmt.Errorf("shutdown http server: %w", err)
		}
		return nil
	case err := <-errorsChannel:
		if err == nil || errors.Is(err, net.ErrClosed) {
			return nil
		}
		return fmt.Errorf("serve http: %w", err)
	}
}
