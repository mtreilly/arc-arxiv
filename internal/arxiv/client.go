// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

package arxiv

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/mtreilly/goarxiv"
)

// Author represents a paper author with optional affiliation.
type Author struct {
	Name        string `yaml:"name"`
	Affiliation string `yaml:"affiliation,omitempty"`
}

// ArxivMeta represents paper metadata stored locally.
type ArxivMeta struct {
	ID              string   `yaml:"id"`
	ArxivID         string   `yaml:"arxiv_id"`
	Title           string   `yaml:"title"`
	SourceType      string   `yaml:"source_type"`
	URL             string   `yaml:"url"`
	PDFURL          string   `yaml:"pdf_url"`
	Published       string   `yaml:"published"`
	Updated         string   `yaml:"updated"`
	Authors         []Author `yaml:"authors"`
	Abstract        string   `yaml:"abstract"`
	Categories      []string `yaml:"categories"`
	PrimaryCategory string   `yaml:"primary_category"`
	Comment         string   `yaml:"comment,omitempty"`
	JournalRef      string   `yaml:"journal_ref,omitempty"`
	DOI             string   `yaml:"doi,omitempty"`
	Version         int      `yaml:"version"`
	FetchedAt       string   `yaml:"fetched_at"`
}

// Client wraps goarxiv.Client with additional functionality.
type Client struct {
	client *goarxiv.Client
}

// NewClient creates a new arxiv client with sensible defaults.
func NewClient() (*Client, error) {
	c, err := goarxiv.New(
		goarxiv.WithUserAgent("arc-arxiv/1.0"),
	)
	if err != nil {
		return nil, fmt.Errorf("create arxiv client: %w", err)
	}
	return &Client{client: c}, nil
}

// FetchArticle retrieves a single article by arXiv ID.
func (c *Client) FetchArticle(ctx context.Context, id string) (*ArxivMeta, error) {
	article, err := c.client.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetch article %s: %w", id, err)
	}
	return articleToMeta(article), nil
}

// FetchArticles retrieves multiple articles by their IDs.
func (c *Client) FetchArticles(ctx context.Context, ids []string) ([]*ArxivMeta, error) {
	articles, err := c.client.GetByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("fetch articles: %w", err)
	}
	metas := make([]*ArxivMeta, 0, len(articles))
	for _, article := range articles {
		metas = append(metas, articleToMeta(article))
	}
	return metas, nil
}

// SearchOptions controls search behavior.
type SearchOptions struct {
	Author     string
	Title      string
	Abstract   string
	Category   string
	MaxResults int
	SortBy     string
}

// Search performs an arXiv search with the given query and options.
func (c *Client) Search(ctx context.Context, query string, opts *SearchOptions) ([]*ArxivMeta, int, error) {
	builder := goarxiv.NewQueryBuilder()

	if query != "" {
		builder.AllFields(query)
	}

	if opts != nil {
		if opts.Author != "" {
			if builder.HasClauses() {
				builder.And()
			}
			builder.Author(opts.Author)
		}
		if opts.Title != "" {
			if builder.HasClauses() {
				builder.And()
			}
			builder.Title(opts.Title)
		}
		if opts.Abstract != "" {
			if builder.HasClauses() {
				builder.And()
			}
			builder.Abstract(opts.Abstract)
		}
		if opts.Category != "" {
			if builder.HasClauses() {
				builder.And()
			}
			builder.Category(opts.Category)
		}
	}

	if !builder.HasClauses() {
		return nil, 0, fmt.Errorf("search query cannot be empty")
	}

	maxResults := 10
	if opts != nil && opts.MaxResults > 0 {
		maxResults = opts.MaxResults
	}

	searchOpts := &goarxiv.SearchOptions{
		MaxResults: maxResults,
		SortBy:     goarxiv.SortByRelevance,
		SortOrder:  goarxiv.SortOrderDescending,
	}

	if opts != nil && opts.SortBy != "" {
		switch opts.SortBy {
		case "relevance":
			searchOpts.SortBy = goarxiv.SortByRelevance
		case "submitted", "submittedDate":
			searchOpts.SortBy = goarxiv.SortBySubmittedDate
		case "updated", "lastUpdatedDate":
			searchOpts.SortBy = goarxiv.SortByLastUpdatedDate
		}
	}

	results, err := c.client.Search(ctx, builder.Build(), searchOpts)
	if err != nil {
		return nil, 0, fmt.Errorf("search: %w", err)
	}

	metas := make([]*ArxivMeta, 0, len(results.Articles))
	for i := range results.Articles {
		metas = append(metas, articleToMeta(&results.Articles[i]))
	}

	return metas, results.TotalResults, nil
}

// DownloadProgress is called during PDF download with progress info.
type DownloadProgress func(downloaded, total int64)

// DownloadPDF downloads the PDF for an article to the specified path.
func (c *Client) DownloadPDF(ctx context.Context, id string, destPath string, progress DownloadProgress) error {
	normalizedID, err := NormalizeArxivID(id)
	if err != nil {
		return fmt.Errorf("invalid arxiv id: %w", err)
	}

	pdfURL := fmt.Sprintf("https://arxiv.org/pdf/%s.pdf", normalizedID)

	req, err := http.NewRequestWithContext(ctx, "GET", pdfURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "arc-arxiv/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()

	if progress != nil && resp.ContentLength > 0 {
		pw := &progressWriter{
			total: resp.ContentLength,
			cb:    progress,
		}
		_, err = io.Copy(io.MultiWriter(f, pw), resp.Body)
	} else {
		_, err = io.Copy(f, resp.Body)
	}

	return err
}

type progressWriter struct {
	total   int64
	written int64
	cb      DownloadProgress
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n := len(p)
	pw.written += int64(n)
	if pw.cb != nil {
		pw.cb(pw.written, pw.total)
	}
	return n, nil
}

// articleToMeta converts a goarxiv.Article to our ArxivMeta format.
func articleToMeta(article *goarxiv.Article) *ArxivMeta {
	if article == nil {
		return nil
	}

	authors := make([]Author, 0, len(article.Authors))
	for _, a := range article.Authors {
		author := Author{Name: a.Name}
		if a.Affiliation != nil {
			author.Affiliation = *a.Affiliation
		}
		authors = append(authors, author)
	}

	baseID := article.BaseID()

	meta := &ArxivMeta{
		ID:              fmt.Sprintf("paper-%s", baseID),
		ArxivID:         baseID,
		Title:           strings.TrimSpace(article.Title),
		SourceType:      "arxiv",
		URL:             article.AbstractURL(),
		PDFURL:          article.PDFURL(),
		Published:       article.Published.Format(time.RFC3339),
		Updated:         article.Updated.Format(time.RFC3339),
		Authors:         authors,
		Abstract:        strings.TrimSpace(article.Summary),
		Categories:      article.Categories,
		PrimaryCategory: article.PrimaryCategory,
		Version:         article.Version(),
		FetchedAt:       time.Now().Format(time.RFC3339),
	}

	if article.Comment != nil {
		meta.Comment = *article.Comment
	}
	if article.JournalRef != nil {
		meta.JournalRef = *article.JournalRef
	}
	if article.DOI != nil {
		meta.DOI = *article.DOI
	}

	return meta
}

// MetaToArticle converts an ArxivMeta back to a goarxiv.Article for export.
func MetaToArticle(meta *ArxivMeta) *goarxiv.Article {
	if meta == nil {
		return nil
	}

	authors := make([]goarxiv.Author, 0, len(meta.Authors))
	for _, a := range meta.Authors {
		author := goarxiv.Author{Name: a.Name}
		if a.Affiliation != "" {
			aff := a.Affiliation
			author.Affiliation = &aff
		}
		authors = append(authors, author)
	}

	published, _ := time.Parse(time.RFC3339, meta.Published)
	updated, _ := time.Parse(time.RFC3339, meta.Updated)

	article := &goarxiv.Article{
		ID:              meta.ArxivID,
		Title:           meta.Title,
		Summary:         meta.Abstract,
		Authors:         authors,
		Published:       published,
		Updated:         updated,
		PrimaryCategory: meta.PrimaryCategory,
		Categories:      meta.Categories,
	}

	if meta.Comment != "" {
		comment := meta.Comment
		article.Comment = &comment
	}
	if meta.JournalRef != "" {
		jr := meta.JournalRef
		article.JournalRef = &jr
	}
	if meta.DOI != "" {
		doi := meta.DOI
		article.DOI = &doi
	}

	return article
}

var (
	oldIDPattern = regexp.MustCompile(`^[a-z-]+/\d{7}(v\d+)?$`)
	newIDPattern = regexp.MustCompile(`^\d{4}\.\d{4,5}(v\d+)?$`)
	urlPattern   = regexp.MustCompile(`arxiv\.org/(?:abs|pdf)/([a-z-]+/\d{7}|\d{4}\.\d{4,5})(v\d+)?(?:\.pdf)?`)
)

// NormalizeArxivID extracts and validates an arXiv ID from various input formats.
func NormalizeArxivID(input string) (string, error) {
	input = strings.TrimSpace(input)

	// Try URL extraction first
	if matches := urlPattern.FindStringSubmatch(input); len(matches) >= 2 {
		id := matches[1]
		if len(matches) > 2 && matches[2] != "" {
			id += matches[2]
		}
		return id, nil
	}

	// Try direct ID validation
	if oldIDPattern.MatchString(input) || newIDPattern.MatchString(input) {
		return input, nil
	}

	return "", fmt.Errorf("invalid arXiv ID or URL: %s", input)
}

// IsValidArxivID checks if the input is a valid arXiv ID.
func IsValidArxivID(input string) bool {
	_, err := NormalizeArxivID(input)
	return err == nil
}
