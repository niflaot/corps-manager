package discord

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

// Client manages a Discord gateway session.
type Client struct {
	session   *discordgo.Session
	log       *zap.Logger
	connected atomic.Bool
}

// New creates a Discord client. An empty token creates a disabled client.
func New(config Config, log *zap.Logger) (*Client, error) {
	if strings.TrimSpace(config.Token) == "" {
		return &Client{log: log}, nil
	}
	session, err := discordgo.New("Bot " + config.Token)
	if err != nil {
		return nil, fmt.Errorf("create discord session: %w", err)
	}
	session.Identify.Intents = config.Intents
	client := &Client{session: session, log: log}
	client.AddHandler(func(_ *discordgo.Session, ready *discordgo.Ready) {
		log.Info("discord gateway ready", zap.String("user", ready.User.String()))
	})
	return client, nil
}

// AddHandler registers a DiscordGo event handler and returns its remover.
func (client *Client) AddHandler(handler any) func() {
	if client.session == nil {
		return func() {}
	}
	return client.session.AddHandler(handler)
}

// SDK returns the underlying DiscordGo session, or nil when disabled.
func (client *Client) SDK() *discordgo.Session {
	return client.session
}

// Enabled reports whether a Discord token configured the gateway.
func (client *Client) Enabled() bool {
	return client.session != nil
}

// Connected reports whether the gateway session is open.
func (client *Client) Connected() bool {
	return client.connected.Load()
}

// Run opens the Discord gateway and blocks until cancellation.
func (client *Client) Run(ctx context.Context) error {
	if client.session == nil {
		client.log.Warn("discord gateway disabled because no token is configured")
		<-ctx.Done()
		return nil
	}
	if err := client.session.Open(); err != nil {
		return fmt.Errorf("open discord gateway: %w", err)
	}
	client.connected.Store(true)
	defer func() {
		client.connected.Store(false)
		if err := client.session.Close(); err != nil {
			client.log.Warn("close discord gateway", zap.Error(err))
		}
	}()
	<-ctx.Done()
	return nil
}
