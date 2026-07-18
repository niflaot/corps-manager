package bootstrap

import (
	"context"
	"sync"

	"github.com/pixelados-net/discord-bot/internal/cronjob"
	"github.com/pixelados-net/discord-bot/internal/messages"
	"github.com/pixelados-net/discord-bot/platform/discord"
	"github.com/pixelados-net/discord-bot/platform/events"
	"github.com/pixelados-net/discord-bot/platform/httpapi"
	"go.uber.org/fx"
	"golang.org/x/sync/errgroup"
)

// Runtime owns concurrently running adapters and workers.
type Runtime struct {
	server     *httpapi.Server
	discord    *discord.Client
	scheduler  *cronjob.Scheduler
	reconciler *messages.Reconciler
	cancel     context.CancelFunc
	done       chan struct{}
	err        error
	mutex      sync.RWMutex
}

func newRuntime(lifecycle fx.Lifecycle, server *httpapi.Server, discordClient *discord.Client,
	scheduler *cronjob.Scheduler, reconciler *messages.Reconciler, _ *events.Bus, _ discord.Handlers) *Runtime {
	runtime := &Runtime{server: server, discord: discordClient, scheduler: scheduler, reconciler: reconciler, done: make(chan struct{})}
	lifecycle.Append(fx.Hook{OnStart: runtime.start, OnStop: runtime.stop})
	return runtime
}

func (runtime *Runtime) start(context.Context) error {
	ctx, cancel := context.WithCancel(context.Background())
	runtime.cancel = cancel
	group, groupContext := errgroup.WithContext(ctx)
	group.Go(func() error { return runtime.server.Run(groupContext) })
	group.Go(func() error { return runtime.discord.Run(groupContext) })
	group.Go(func() error { return runtime.scheduler.Run(groupContext) })
	group.Go(func() error { return runtime.reconciler.Run(groupContext) })
	go func() {
		err := group.Wait()
		runtime.mutex.Lock()
		runtime.err = err
		runtime.mutex.Unlock()
		close(runtime.done)
	}()
	return nil
}

func (runtime *Runtime) stop(ctx context.Context) error {
	if runtime.cancel != nil {
		runtime.cancel()
	}
	select {
	case <-runtime.done:
		return runtime.Err()
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Done closes when any running adapter fails or shutdown completes.
func (runtime *Runtime) Done() <-chan struct{} { return runtime.done }

// Err returns the asynchronous runtime result after Done closes.
func (runtime *Runtime) Err() error {
	runtime.mutex.RLock()
	defer runtime.mutex.RUnlock()
	return runtime.err
}
