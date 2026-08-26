package discord

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync/atomic"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

var snowflakePattern = regexp.MustCompile(`^[0-9]{1,20}$`)

const discordLoggerName = "discordgo"

// Client manages a Discord gateway session.
type Client struct {
	session   *discordgo.Session
	log       *zap.Logger
	connected atomic.Bool
	userID    atomic.Value
	guildID   string
}

// New creates the configured Discord bot client without opening its gateway.
func New(config Config, log *zap.Logger) (*Client, error) {
	session, err := discordgo.New("Bot " + config.Token)
	if err != nil {
		return nil, fmt.Errorf("create discord session: %w", err)
	}
	session.Identify.Intents = config.Intents
	configureDiscordLogger(session, log)
	client := &Client{session: session, log: log, guildID: config.GuildID}
	client.AddHandler(func(_ *discordgo.Session, ready *discordgo.Ready) {
		client.userID.Store(ready.User.ID)
		log.Info("discord gateway ready", zap.String("user", ready.User.String()))
	})
	return client, nil
}

func configureDiscordLogger(session *discordgo.Session, log *zap.Logger) {
	switch {
	case log.Core().Enabled(zap.DebugLevel):
		session.LogLevel = discordgo.LogDebug
	case log.Core().Enabled(zap.InfoLevel):
		session.LogLevel = discordgo.LogInformational
	case log.Core().Enabled(zap.WarnLevel):
		session.LogLevel = discordgo.LogWarning
	default:
		session.LogLevel = discordgo.LogError
	}
	discordLog := log.Named(discordLoggerName).WithOptions(zap.AddCallerSkip(2))
	discordgo.Logger = func(level, _ int, format string, arguments ...interface{}) {
		message := strings.TrimSpace(fmt.Sprintf(format, arguments...))
		field := zap.Int("discordgo_level", level)
		switch level {
		case discordgo.LogError:
			discordLog.Error(message, field)
		case discordgo.LogWarning:
			discordLog.Warn(message, field)
		case discordgo.LogInformational:
			discordLog.Info(message, field)
		default:
			discordLog.Debug(message, field)
		}
	}
}

// GuildID returns the only configured Discord guild snowflake.
func (client *Client) GuildID() string {
	return client.guildID
}

// ValidateAuthentication verifies the bot token before workers start.
func (client *Client) ValidateAuthentication(ctx context.Context) error {
	if _, err := client.BotUserID(ctx); err != nil {
		return fmt.Errorf("authenticate Discord bot: %w", err)
	}
	return nil
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
