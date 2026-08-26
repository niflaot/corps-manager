// Package bootstrap composes and runs the application through Uber Fx.
package bootstrap

import (
	"context"
	"fmt"
	"sync"
	"time"

	appconfig "github.com/niflaot/corps-manager/platform/app"
	"go.uber.org/fx"
)

const shutdownTimeout = 15 * time.Second

// Application owns the Fx graph and running process lifecycle.
type Application struct {
	app       *fx.App
	runtime   *Runtime
	closeOnce sync.Once
}

// New builds and starts the Fx application graph.
func New(ctx context.Context, version string) (*Application, error) {
	var runtime *Runtime
	fxApp := fx.New(
		Module,
		fx.Supply(appconfig.Version(version)),
		fx.Populate(&runtime),
		fx.NopLogger,
	)
	if err := fxApp.Start(ctx); err != nil {
		stopContext, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		_ = fxApp.Stop(stopContext)
		return nil, fmt.Errorf("start application: %w", err)
	}
	return &Application{app: fxApp, runtime: runtime}, nil
}

// Run waits for cancellation or an asynchronous runtime failure.
func (application *Application) Run(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return nil
	case <-application.runtime.Done():
		return application.runtime.Err()
	}
}

// Close stops the Fx graph once in reverse dependency order.
func (application *Application) Close() {
	application.closeOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		_ = application.app.Stop(ctx)
	})
}
