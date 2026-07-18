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

// ValidateGuildAdministrator verifies exclusive guild membership and administrator permission.
func (client *Client) ValidateGuildAdministrator(ctx context.Context) error {
	userID, err := client.BotUserID(ctx)
	if err != nil {
		return fmt.Errorf("authenticate Discord bot: %w", err)
	}
	guilds, err := client.session.UserGuilds(200, "", "", false, discordgo.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("list Discord guilds: %w", err)
	}
	if len(guilds) != 1 || guilds[0].ID != client.guildID {
		return fmt.Errorf("discord bot must belong exclusively to configured guild %s", client.guildID)
	}
	member, err := client.session.GuildMember(client.guildID, userID, discordgo.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("read configured guild membership: %w", err)
	}
	roles, err := client.session.GuildRoles(client.guildID, discordgo.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("read configured guild roles: %w", err)
	}
	memberRoles := make(map[string]struct{}, len(member.Roles)+1)
	memberRoles[client.guildID] = struct{}{}
	for _, roleID := range member.Roles {
		memberRoles[roleID] = struct{}{}
	}
	var permissions int64
	for _, role := range roles {
		if _, ok := memberRoles[role.ID]; ok {
			permissions |= role.Permissions
		}
	}
	if permissions&discordgo.PermissionAdministrator == 0 {
		return fmt.Errorf("discord bot requires administrator permission in guild %s", client.guildID)
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
