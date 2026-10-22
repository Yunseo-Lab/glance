package dashboard

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type GitHubSource struct {
	Repository string `json:"repository"`
	View       string `json:"view"`
}

var githubRepository = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]{0,38}/[A-Za-z0-9_.-]{1,100}$`)

func validateGitHubSource(data json.RawMessage) error {
	var config GitHubSource
	if err := decodeConfig(data, &config); err != nil {
		return err
	}
	parts := strings.Split(config.Repository, "/")
	if !githubRepository.MatchString(config.Repository) || len(parts) != 2 || parts[1] == "." || parts[1] == ".." {
		return ErrInvalidInput
	}
	switch config.View {
	case "repository", "releases", "issues", "pulls":
		return nil
	}
	return ErrInvalidInput
}
func (p *Providers) fetchGitHub(ctx context.Context, source Source) (any, error) {
	var config GitHubSource
	if err := decodeConfig(source.Config, &config); err != nil {
		return nil, err
	}
	path := "https://api.github.com/repos/" + config.Repository
	if config.View != "repository" {
		path += "/" + config.View + "?per_page=20&state=open"
	}
	headers := http.Header{}
	if source.ConnectionID != "" {
		if p.store == nil {
			return nil, ErrConnectionLocked
		}
		connection, secret, err := p.store.resolveConnection(source.ConnectionID)
		if err != nil || connection.Kind != "github" {
			return nil, &ProviderError{Code: "authentication"}
		}
		headers.Set("Authorization", "Bearer "+secret.Token)
	}
	data, err := p.getWithHeaders(ctx, path, headers)
	if err != nil {
		return nil, err
	}
	return parseGitHub(data, config)
}
func parseGitHub(data []byte, config GitHubSource) (FeedData, error) {
	out := FeedData{Title: config.Repository, Items: []FeedItem{}}
	add := func(identity, title, address, excerpt string, at *time.Time) {
		link, err := canonicalArticleURL(address, "https://github.com/"+config.Repository)
		if err != nil {
			return
		}
		parsed, _ := url.Parse(link)
		if parsed.Hostname() != "github.com" {
			return
		}
		sum := sha256.Sum256([]byte("github:" + config.Repository + ":" + identity))
		out.Items = append(out.Items, FeedItem{ID: hex.EncodeToString(sum[:]), Title: plainText(title, 300), URL: link, Excerpt: plainText(excerpt, 600), PublishedAt: at})
	}
	if config.View == "repository" {
		var repo struct {
			FullName    string     `json:"full_name"`
			HTMLURL     string     `json:"html_url"`
			Description string     `json:"description"`
			Stars       int        `json:"stargazers_count"`
			Forks       int        `json:"forks_count"`
			OpenIssues  int        `json:"open_issues_count"`
			UpdatedAt   *time.Time `json:"updated_at"`
		}
		if err := json.Unmarshal(data, &repo); err != nil || repo.FullName == "" {
			return out, &ProviderError{Code: "invalid_response"}
		}
		add("repository", repo.FullName, repo.HTMLURL, fmt.Sprintf("%s · ★ %d · 포크 %d · 열린 이슈/PR %d", repo.Description, repo.Stars, repo.Forks, repo.OpenIssues), repo.UpdatedAt)
		return out, nil
	}
	var entries []struct {
		ID          int64           `json:"id"`
		Title       string          `json:"title"`
		Name        string          `json:"name"`
		Tag         string          `json:"tag_name"`
		HTMLURL     string          `json:"html_url"`
		Body        string          `json:"body"`
		PublishedAt *time.Time      `json:"published_at"`
		UpdatedAt   *time.Time      `json:"updated_at"`
		Draft       bool            `json:"draft"`
		PullRequest json.RawMessage `json:"pull_request"`
	}
	if err := json.Unmarshal(data, &entries); err != nil {
		return out, &ProviderError{Code: "invalid_response"}
	}
	for _, entry := range entries {
		if entry.Draft || config.View == "issues" && len(entry.PullRequest) > 0 {
			continue
		}
		title := entry.Title
		if config.View == "releases" {
			title = entry.Name
			if title == "" {
				title = entry.Tag
			}
		}
		at := entry.PublishedAt
		if at == nil {
			at = entry.UpdatedAt
		}
		add(fmt.Sprint(entry.ID), title, entry.HTMLURL, entry.Body, at)
		if len(out.Items) == 20 {
			break
		}
	}
	return out, nil
}
