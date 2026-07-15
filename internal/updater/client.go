package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultGitHubAPIBaseURL = "https://api.github.com"
	gitHubAcceptHeader      = "application/vnd.github.v3+json"
)

var defaultHTTPClient = &http.Client{Timeout: 30 * time.Second}

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func (c Client) Latest(ctx context.Context) (Release, error) {
	var release Release
	if err := c.getJSON(ctx, "/repos/tcp404/OneTiny/releases/latest", &release); err != nil {
		return Release{}, err
	}
	return release, nil
}

func (c Client) ByTag(ctx context.Context, tag string) (Release, error) {
	var release Release
	path := "/repos/tcp404/OneTiny/releases/tags/" + url.PathEscape(tag)
	if err := c.getJSON(ctx, path, &release); err != nil {
		return Release{}, err
	}
	return release, nil
}

func (c Client) Tags(ctx context.Context) ([]string, error) {
	var response []struct {
		Name string `json:"name"`
	}
	if err := c.getJSON(ctx, "/repos/tcp404/OneTiny/tags", &response); err != nil {
		return nil, err
	}

	tags := make([]string, 0, len(response))
	for _, tag := range response {
		tags = append(tags, tag.Name)
	}
	return tags, nil
}

func (c Client) getJSON(ctx context.Context, path string, out any) error {
	endpoint := strings.TrimRight(c.baseURL(), "/") + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("create github request: %w", err)
	}
	req.Header.Set("Accept", gitHubAcceptHeader)

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return fmt.Errorf("github request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		if len(body) > 0 {
			return fmt.Errorf("github request failed: %s: %s", resp.Status, strings.TrimSpace(string(body)))
		}
		return fmt.Errorf("github request failed: %s", resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode github response: %w", err)
	}
	return nil
}

func (c Client) baseURL() string {
	if strings.TrimSpace(c.BaseURL) == "" {
		return defaultGitHubAPIBaseURL
	}
	return c.BaseURL
}

func (c Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return defaultHTTPClient
}
