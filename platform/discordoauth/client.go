package discordoauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/pixelados-net/discord-bot/internal/discordlinks"
	"go.uber.org/zap"
)

const (
	discordAPIBase       = "https://discord.com/api/v10"
	discordAuthorizeURL  = "https://discord.com/oauth2/authorize"
	maximumResponseBytes = 1 << 20
	userAgent            = "DiscordBot (https://github.com/pixelados-net/discord-bot, 1.0)"
)

// Client performs Discord OAuth authorization-code operations.
type Client struct {
	config       Config
	http         *http.Client
	log          *zap.Logger
	apiBase      string
	authorizeURL string
}

// New creates a bounded Discord OAuth HTTP client.
func New(config Config, log *zap.Logger) *Client {
	return &Client{config: config, http: &http.Client{Timeout: config.HTTPTimeout}, log: log,
		apiBase: discordAPIBase, authorizeURL: discordAuthorizeURL}
}

// Enabled reports whether Discord OAuth has complete runtime configuration.
func (client *Client) Enabled() bool { return client.config.Enabled }

// AuthorizationURL builds Discord's authorization endpoint URL.
func (client *Client) AuthorizationURL(state string) string {
	query := url.Values{
		"client_id":     {client.config.ClientID},
		"redirect_uri":  {client.config.CallbackURL()},
		"response_type": {"code"},
		"scope":         {"identify"},
		"state":         {state},
	}
	return client.authorizeURL + "?" + query.Encode()
}

// Exchange exchanges one authorization code for a transient user grant.
func (client *Client) Exchange(ctx context.Context, code string) (discordlinks.AccessGrant, error) {
	form := url.Values{"grant_type": {"authorization_code"}, "code": {code},
		"redirect_uri": {client.config.CallbackURL()}}
	request, err := client.formRequest(ctx, client.apiBase+"/oauth2/token", form)
	if err != nil {
		return discordlinks.AccessGrant{}, err
	}
	var response struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
	}
	if err = client.executeJSON(request, &response); err != nil {
		return discordlinks.AccessGrant{}, fmt.Errorf("%w: exchange authorization code", discordlinks.ErrProvider)
	}
	if response.AccessToken == "" || !strings.EqualFold(response.TokenType, "Bearer") {
		return discordlinks.AccessGrant{}, fmt.Errorf("%w: invalid token response", discordlinks.ErrProvider)
	}
	return discordlinks.AccessGrant{AccessToken: response.AccessToken, TokenType: response.TokenType,
		Scope: response.Scope}, nil
}

// CurrentUser resolves the Discord identity represented by a grant.
func (client *Client) CurrentUser(ctx context.Context, grant discordlinks.AccessGrant) (discordlinks.Identity, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, client.apiBase+"/users/@me", nil)
	if err != nil {
		return discordlinks.Identity{}, fmt.Errorf("create current user request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+grant.AccessToken)
	request.Header.Set("User-Agent", userAgent)
	var response struct {
		ID         string `json:"id"`
		Username   string `json:"username"`
		GlobalName string `json:"global_name"`
		Avatar     string `json:"avatar"`
		Bot        bool   `json:"bot"`
	}
	if err = client.executeJSON(request, &response); err != nil {
		return discordlinks.Identity{}, fmt.Errorf("%w: read current user", discordlinks.ErrProvider)
	}
	return discordlinks.Identity{UserID: response.ID, Username: response.Username,
		GlobalName: response.GlobalName, AvatarHash: response.Avatar, Bot: response.Bot}, nil
}

// Revoke invalidates the transient Discord authorization.
func (client *Client) Revoke(ctx context.Context, grant discordlinks.AccessGrant) error {
	form := url.Values{"token": {grant.AccessToken}, "token_type_hint": {"access_token"}}
	request, err := client.formRequest(ctx, client.apiBase+"/oauth2/token/revoke", form)
	if err == nil {
		var response any
		err = client.executeJSONOptional(request, &response)
	}
	if err != nil {
		client.log.Error("discord oauth token revocation failed", zap.Error(err))
		return fmt.Errorf("%w: revoke authorization", discordlinks.ErrProvider)
	}
	client.log.Debug("discord oauth token revoked")
	return nil
}

func (client *Client) formRequest(ctx context.Context, endpoint string, values url.Values) (*http.Request, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create OAuth request: %w", err)
	}
	request.SetBasicAuth(client.config.ClientID, client.config.ClientSecret)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("User-Agent", userAgent)
	return request, nil
}

func (client *Client) executeJSON(request *http.Request, target any) error {
	return client.execute(request, func(reader io.Reader) error { return json.NewDecoder(reader).Decode(target) })
}

func (client *Client) executeJSONOptional(request *http.Request, target any) error {
	return client.execute(request, func(reader io.Reader) error {
		err := json.NewDecoder(reader).Decode(target)
		if err == io.EOF {
			return nil
		}
		return err
	})
}

func (client *Client) execute(request *http.Request, decode func(io.Reader) error) error {
	response, err := client.http.Do(request)
	if err != nil {
		client.log.Error("discord oauth request failed", zap.Error(err))
		return err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		client.log.Error("discord oauth response failed", zap.Int("status", response.StatusCode))
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, maximumResponseBytes))
		return fmt.Errorf("discord returned HTTP %d", response.StatusCode)
	}
	return decode(io.LimitReader(response.Body, maximumResponseBytes))
}
