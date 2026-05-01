package usecase

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Nanyak/thangdq-lab/internal/entity"
	"github.com/Nanyak/thangdq-lab/pkg/errors"
	"golang.org/x/net/html"
)

type URLShortener struct {
	repo      URLRepository
	cache     URLCache
	generator ShortCodeGenerator
}

func NewURLShortener(repo URLRepository, cache URLCache, generator ShortCodeGenerator) *URLShortener {
	return &URLShortener{
		repo:      repo,
		cache:     cache,
		generator: generator,
	}
}

func (u *URLShortener) CreateShortURL(ctx context.Context, originalURL string, userID string) (*entity.Link, error) {
	shortCode := u.generator.Generate(originalURL)

	// Check cache first - if exists and anonymous request, return existing link
	if userID == "" {
		if url, err := u.cache.Get(ctx, shortCode); err == nil {
			return &entity.Link{
				ShortCode:   shortCode,
				OriginalURL: url,
			}, nil
		}
	}

	title := fetchTitle(originalURL)

	link := &entity.Link{
		ShortCode:   shortCode,
		OriginalURL: originalURL,
		UserID:      userID,
		Title:       title,
		CreatedAt:   time.Now(),
	}

	// Save to MongoDB
	if err := u.repo.Save(ctx, link); err != nil {
		// If duplicate, fetch existing and return it
		if errors.IsDuplicate(err) {
			existing, fetchErr := u.repo.FindByShortCode(ctx, shortCode)
			if fetchErr != nil {
				return nil, fmt.Errorf("failed to fetch existing link: %w", fetchErr)
			}
			// Cache existing link
			_ = u.cache.Set(ctx, shortCode, existing.OriginalURL)
			return existing, nil
		}
		return nil, fmt.Errorf("failed to save link: %w", err)
	}

	// Cache in Redis (best effort)
	if err := u.cache.Set(ctx, shortCode, originalURL); err != nil {
		fmt.Printf("Warning: failed to cache link: %v\n", err)
	}

	return link, nil
}

func (u *URLShortener) GetOriginalURL(ctx context.Context, shortCode string) (string, error) {
	// Try cache first
	if url, err := u.cache.Get(ctx, shortCode); err == nil {
		return url, nil
	}

	// Cache miss, fetch from MongoDB
	link, err := u.repo.FindByShortCode(ctx, shortCode)
	if err != nil {
		return "", fmt.Errorf("failed to find link: %w", err)
	}

	// Populate cache (best effort)
	if err := u.cache.Set(ctx, shortCode, link.OriginalURL); err != nil {
		fmt.Printf("Warning: failed to populate cache: %v\n", err)
	}

	return link.OriginalURL, nil
}

func (u *URLShortener) GetUserLinks(ctx context.Context, userID string) ([]*entity.Link, error) {
	links, err := u.repo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user links: %w", err)
	}
	return links, nil
}

// fetchTitle fetches the page title from og:title or <title> with a 5s timeout.
// Returns empty string on any failure.
func fetchTitle(rawURL string) string {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(rawURL)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	// Limit body read to 64KB to avoid huge pages
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return ""
	}

	return parseTitle(string(body))
}

func parseTitle(body string) string {
	// Try og:title first
	if title := extractMetaContent(body, "og:title"); title != "" {
		return title
	}
	// Fallback to <title> tag
	doc, err := html.Parse(strings.NewReader(body))
	if err != nil {
		return ""
	}
	return findTitleTag(doc)
}

func extractMetaContent(body, property string) string {
	doc, err := html.Parse(strings.NewReader(body))
	if err != nil {
		return ""
	}
	return findMetaContent(doc, property)
}

func findMetaContent(n *html.Node, property string) string {
	if n.Type == html.ElementNode && n.Data == "meta" {
		var prop, content string
		for _, a := range n.Attr {
			switch a.Key {
			case "property", "name":
				prop = a.Val
			case "content":
				content = a.Val
			}
		}
		if prop == property {
			return content
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if result := findMetaContent(c, property); result != "" {
			return result
		}
	}
	return ""
}

func findTitleTag(n *html.Node) string {
	if n.Type == html.ElementNode && n.Data == "title" {
		if n.FirstChild != nil {
			return strings.TrimSpace(n.FirstChild.Data)
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if result := findTitleTag(c); result != "" {
			return result
		}
	}
	return ""
}
