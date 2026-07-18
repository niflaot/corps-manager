package discord

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

// Client manages a Discord gateway session.
type Client struct {
	session   *discordgo.Session
	log       *zap.Logger
	connected atomic.Bool
	userID    atomic.Value
}

// New creates the configured Discord bot client without opening its gateway.
func New(config Config, log *zap.Logger) (*Client, error) {
	session, err := discordgo.New("Bot " + config.Token)
	if err != nil {
		return nil, fmt.Errorf("create discord session: %w", err)
	}
	session.Identify.Intents = config.Intents
	client := &Client{session: session, log: log}
	client.AddHandler(func(_ *discordgo.Session, ready *discordgo.Ready) {
		client.userID.Store(ready.User.ID)
		log.Info("discord gateway ready", zap.String("user", ready.User.String()))
	})
	return client, nil
}

// BotUserID returns the authenticated Discord bot user snowflake.
func (client *Client) BotUserID(ctx context.Context) (string, error) {
	if stored := client.userID.Load(); stored != nil {
		return stored.(string), nil
	}
	user, err := client.session.User("@me", discordgo.WithContext(ctx))
	if err != nil {
		return "", err
	}
	client.userID.Store(user.ID)
	return user.ID, nil
}

// AddHandler registers a DiscordGo event handler and returns its remover.
func (client *Client) AddHandler(handler any) func() {
	return client.session.AddHandler(handler)
}

// SDK returns the underlying DiscordGo session.
func (client *Client) SDK() *discordgo.Session {
	return client.session
}

// Connected reports whether the gateway session is open.
func (client *Client) Connected() bool {
	return client.connected.Load()
}

// Run opens the Discord gateway and blocks until cancellation.
func (client *Client) Run(ctx context.Context) error {
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
