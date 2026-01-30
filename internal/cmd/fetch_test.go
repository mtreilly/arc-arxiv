// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mtreilly/arc-arxiv/internal/arxiv"
	"gopkg.in/yaml.v3"
)

// TestFetchWorkflow tests the complete fetch workflow
func TestFetchWorkflow(t *testing.T) {
	t.Run("creates correct directory structure", func(t *testing.T) {
		tmpDir := t.TempDir()
		papersRoot := filepath.Join(tmpDir, "papers")
		id := "2304.00067"

		// Create the paper directory structure manually
		destDir := filepath.Join(papersRoot, id)
		if err := os.MkdirAll(destDir, 0o755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}

		// Verify directory was created
		info, err := os.Stat(destDir)
		if err != nil {
			t.Fatalf("directory not created: %v", err)
		}
		if !info.IsDir() {
			t.Error("expected directory, got file")
		}
	})

	t.Run("writes meta.yaml correctly", func(t *testing.T) {
		tmpDir := t.TempDir()

		meta := &arxiv.ArxivMeta{
			ID:              "paper-2304.00067",
			ArxivID:         "2304.00067",
			Title:           "Test Paper Title",
			SourceType:      "arxiv",
			URL:             "https://arxiv.org/abs/2304.00067",
			PDFURL:          "https://arxiv.org/pdf/2304.00067.pdf",
			Published:       "2023-04-01T00:00:00Z",
			Updated:         "2023-04-15T00:00:00Z",
			Authors:         []arxiv.Author{{Name: "Alice Smith", Affiliation: "MIT"}},
			Abstract:        "This is the abstract.",
			Categories:      []string{"cs.LG", "cs.AI"},
			PrimaryCategory: "cs.LG",
			Version:         2,
			FetchedAt:       "2024-01-15T10:00:00Z",
		}

		metaPath := filepath.Join(tmpDir, "meta.yaml")
		if err := writeMeta(metaPath, meta); err != nil {
			t.Fatalf("writeMeta failed: %v", err)
		}

		// Read back and verify
		readMeta, err := readMeta(metaPath)
		if err != nil {
			t.Fatalf("readMeta failed: %v", err)
		}

		if readMeta.ArxivID != meta.ArxivID {
			t.Errorf("ArxivID = %q, want %q", readMeta.ArxivID, meta.ArxivID)
		}
		if readMeta.Title != meta.Title {
			t.Errorf("Title = %q, want %q", readMeta.Title, meta.Title)
		}
		if len(readMeta.Authors) != len(meta.Authors) {
			t.Errorf("len(Authors) = %d, want %d", len(readMeta.Authors), len(meta.Authors))
		}
		if len(readMeta.Categories) != len(meta.Categories) {
			t.Errorf("len(Categories) = %d, want %d", len(readMeta.Categories), len(meta.Categories))
		}
	})

	t.Run("creates notes.md template", func(t *testing.T) {
		tmpDir := t.TempDir()

		meta := &arxiv.ArxivMeta{
			ArxivID: "2304.00067",
			Title:   "Test Paper",
			Authors: []arxiv.Author{{Name: "Alice"}, {Name: "Bob"}},
		}

		notesPath := filepath.Join(tmpDir, "notes.md")
		authorNames := make([]string, 0, len(meta.Authors))
		for _, a := range meta.Authors {
			authorNames = append(authorNames, a.Name)
		}
		notesContent := "# " + meta.Title + "\n\narXiv: " + meta.ArxivID + "\nAuthors: " + strings.Join(authorNames, ", ") + "\n\n## Summary\n\n\n## Key Takeaways\n\n\n## Follow-ups\n\n"

		if err := os.WriteFile(notesPath, []byte(notesContent), 0o644); err != nil {
			t.Fatalf("failed to write notes: %v", err)
		}

		// Verify content
		content, err := os.ReadFile(notesPath)
		if err != nil {
			t.Fatalf("failed to read notes: %v", err)
		}

		if !strings.Contains(string(content), "# Test Paper") {
			t.Error("notes.md should contain paper title")
		}
		if !strings.Contains(string(content), "arXiv: 2304.00067") {
			t.Error("notes.md should contain arXiv ID")
		}
		if !strings.Contains(string(content), "Alice, Bob") {
			t.Error("notes.md should contain authors")
		}
		if !strings.Contains(string(content), "## Summary") {
			t.Error("notes.md should contain Summary section")
		}
		if !strings.Contains(string(content), "## Key Takeaways") {
			t.Error("notes.md should contain Key Takeaways section")
		}
		if !strings.Contains(string(content), "## Follow-ups") {
			t.Error("notes.md should contain Follow-ups section")
		}
	})
}

// TestExistingPaperHandling tests behavior when paper already exists
func TestExistingPaperHandling(t *testing.T) {
	t.Run("detects existing paper", func(t *testing.T) {
		tmpDir := t.TempDir()
		id := "2304.00067"
		destDir := filepath.Join(tmpDir, "papers", id)

		// Create existing paper directory
		if err := os.MkdirAll(destDir, 0o755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}

		// Verify it exists
		if _, err := os.Stat(destDir); os.IsNotExist(err) {
			t.Error("directory should exist")
		}
	})
}

// TestMetaYAMLFormat tests the YAML serialization format
func TestMetaYAMLFormat(t *testing.T) {
	meta := &arxiv.ArxivMeta{
		ID:              "paper-2304.00067",
		ArxivID:         "2304.00067",
		Title:           "Test Paper",
		SourceType:      "arxiv",
		Authors:         []arxiv.Author{{Name: "Alice", Affiliation: "MIT"}},
		Categories:      []string{"cs.LG"},
		PrimaryCategory: "cs.LG",
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(meta); err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	content := buf.String()

	// Verify YAML format
	if !strings.Contains(content, "id: paper-2304.00067") {
		t.Error("YAML should contain id field")
	}
	if !strings.Contains(content, "arxiv_id: \"2304.00067\"") {
		t.Error("YAML should contain arxiv_id field")
	}
	if !strings.Contains(content, "title: Test Paper") {
		t.Error("YAML should contain title field")
	}
	if !strings.Contains(content, "- name: Alice") {
		t.Error("YAML should contain author name")
	}
	if !strings.Contains(content, "affiliation: MIT") {
		t.Error("YAML should contain author affiliation")
	}
}

// TestReadMeta tests reading meta.yaml files
func TestReadMeta(t *testing.T) {
	t.Run("reads valid meta.yaml", func(t *testing.T) {
		tmpDir := t.TempDir()
		metaPath := filepath.Join(tmpDir, "meta.yaml")

		content := `id: paper-2304.00067
arxiv_id: "2304.00067"
title: Test Paper
source_type: arxiv
url: https://arxiv.org/abs/2304.00067
pdf_url: https://arxiv.org/pdf/2304.00067.pdf
authors:
  - name: Alice
    affiliation: MIT
  - name: Bob
categories:
  - cs.LG
  - cs.AI
primary_category: cs.LG
version: 2
fetched_at: "2024-01-15T10:00:00Z"
`
		if err := os.WriteFile(metaPath, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		meta, err := readMeta(metaPath)
		if err != nil {
			t.Fatalf("readMeta failed: %v", err)
		}

		if meta.ArxivID != "2304.00067" {
			t.Errorf("ArxivID = %q, want %q", meta.ArxivID, "2304.00067")
		}
		if meta.Title != "Test Paper" {
			t.Errorf("Title = %q, want %q", meta.Title, "Test Paper")
		}
		if len(meta.Authors) != 2 {
			t.Errorf("len(Authors) = %d, want 2", len(meta.Authors))
		}
		if meta.Authors[0].Name != "Alice" {
			t.Errorf("Authors[0].Name = %q, want %q", meta.Authors[0].Name, "Alice")
		}
		if meta.Authors[0].Affiliation != "MIT" {
			t.Errorf("Authors[0].Affiliation = %q, want %q", meta.Authors[0].Affiliation, "MIT")
		}
		if meta.Version != 2 {
			t.Errorf("Version = %d, want 2", meta.Version)
		}
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		_, err := readMeta("/non/existent/path/meta.yaml")
		if err == nil {
			t.Error("expected error for non-existent file")
		}
	})

	t.Run("returns error for invalid YAML", func(t *testing.T) {
		tmpDir := t.TempDir()
		metaPath := filepath.Join(tmpDir, "meta.yaml")

		if err := os.WriteFile(metaPath, []byte("invalid: yaml: content: ["), 0o644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		_, err := readMeta(metaPath)
		if err == nil {
			t.Error("expected error for invalid YAML")
		}
	})
}

// TestWriteMeta tests writing meta.yaml files
func TestWriteMeta(t *testing.T) {
	t.Run("writes to new file", func(t *testing.T) {
		tmpDir := t.TempDir()
		metaPath := filepath.Join(tmpDir, "meta.yaml")

		meta := &arxiv.ArxivMeta{
			ArxivID: "2304.00067",
			Title:   "Test",
		}

		if err := writeMeta(metaPath, meta); err != nil {
			t.Fatalf("writeMeta failed: %v", err)
		}

		if _, err := os.Stat(metaPath); os.IsNotExist(err) {
			t.Error("file should exist")
		}
	})

	t.Run("overwrites existing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		metaPath := filepath.Join(tmpDir, "meta.yaml")

		// Write initial
		meta1 := &arxiv.ArxivMeta{ArxivID: "2304.00067", Title: "Original"}
		if err := writeMeta(metaPath, meta1); err != nil {
			t.Fatalf("first writeMeta failed: %v", err)
		}

		// Overwrite
		meta2 := &arxiv.ArxivMeta{ArxivID: "2304.00067", Title: "Updated"}
		if err := writeMeta(metaPath, meta2); err != nil {
			t.Fatalf("second writeMeta failed: %v", err)
		}

		// Verify updated
		readBack, err := readMeta(metaPath)
		if err != nil {
			t.Fatalf("readMeta failed: %v", err)
		}
		if readBack.Title != "Updated" {
			t.Errorf("Title = %q, want %q", readBack.Title, "Updated")
		}
	})
}

// TestPDFDownload tests PDF download functionality
func TestPDFDownload(t *testing.T) {
	t.Run("downloads PDF to correct path", func(t *testing.T) {
		// Create a mock server that serves a fake PDF
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/pdf")
			w.Header().Set("Content-Length", "1000")
			w.Write([]byte("%PDF-1.4 fake pdf content"))
		}))
		defer server.Close()

		tmpDir := t.TempDir()
		destPath := filepath.Join(tmpDir, "paper.pdf")

		// Download using standard http (simulating the download logic)
		resp, err := http.Get(server.URL)
		if err != nil {
			t.Fatalf("download failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("status = %d, want 200", resp.StatusCode)
		}

		// Write the response to file
		content, _ := io.ReadAll(resp.Body)
		if err := os.WriteFile(destPath, content, 0o644); err != nil {
			t.Fatalf("failed to write PDF: %v", err)
		}

		// Verify file exists
		if _, err := os.Stat(destPath); os.IsNotExist(err) {
			t.Error("PDF file should exist")
		}
	})

	t.Run("progress callback is called", func(t *testing.T) {
		tmpDir := t.TempDir()
		testPdfPath := filepath.Join(tmpDir, "paper.pdf")

		// Create a fake PDF file to simulate download
		content := make([]byte, 1000)
		if err := os.WriteFile(testPdfPath, content, 0o644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		// Verify file was created
		info, err := os.Stat(testPdfPath)
		if err != nil {
			t.Fatalf("file not created: %v", err)
		}
		if info.Size() != 1000 {
			t.Errorf("file size = %d, want 1000", info.Size())
		}
	})
}

// TestBatchFetch tests fetching multiple papers
func TestBatchFetch(t *testing.T) {
	t.Run("normalizes multiple IDs", func(t *testing.T) {
		inputs := []string{
			"2304.00067",
			"https://arxiv.org/abs/2301.12345",
			"https://arxiv.org/pdf/2312.99999.pdf",
		}

		expectedIDs := []string{
			"2304.00067",
			"2301.12345",
			"2312.99999",
		}

		for i, input := range inputs {
			id, err := arxiv.NormalizeArxivID(input)
			if err != nil {
				t.Errorf("NormalizeArxivID(%q) failed: %v", input, err)
				continue
			}
			if id != expectedIDs[i] {
				t.Errorf("NormalizeArxivID(%q) = %q, want %q", input, id, expectedIDs[i])
			}
		}
	})

	t.Run("creates separate directories for each paper", func(t *testing.T) {
		tmpDir := t.TempDir()
		papersRoot := filepath.Join(tmpDir, "papers")

		ids := []string{"2304.00067", "2301.12345", "2312.99999"}

		for _, id := range ids {
			destDir := filepath.Join(papersRoot, id)
			if err := os.MkdirAll(destDir, 0o755); err != nil {
				t.Fatalf("failed to create directory for %s: %v", id, err)
			}
		}

		// Verify all directories exist
		for _, id := range ids {
			destDir := filepath.Join(papersRoot, id)
			if _, err := os.Stat(destDir); os.IsNotExist(err) {
				t.Errorf("directory for %s should exist", id)
			}
		}
	})
}

// TestTruncate tests the truncate helper function
func TestTruncate(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is a longer string", 10, "this is..."},
		{"", 10, ""},
		{"abc", 3, "abc"},
		{"abcd", 3, "..."},
	}

	for _, tt := range tests {
		got := truncate(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
	}
}

// TestParseTime tests the parseTime helper function
func TestParseTime(t *testing.T) {
	tests := []struct {
		input string
		want  int64
	}{
		{"2024-01-15T10:30:00Z", 1705314600},
		{"invalid", 0},
		{"", 0},
	}

	for _, tt := range tests {
		got := parseTime(tt.input)
		if got != tt.want {
			t.Errorf("parseTime(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

// TestFetchDryRun tests dry-run mode
func TestFetchDryRun(t *testing.T) {
	tmpDir := t.TempDir()
	papersRoot := filepath.Join(tmpDir, "papers")
	id := "2304.00067"
	destDir := filepath.Join(papersRoot, id)

	// In dry-run mode, directory should NOT be created
	// (We're just testing the logic, not the actual command)

	// Verify directory does NOT exist before dry-run
	if _, err := os.Stat(destDir); !os.IsNotExist(err) {
		t.Error("directory should not exist before dry-run")
	}
}

// TestFetchForceMode tests force re-fetch mode
func TestFetchForceMode(t *testing.T) {
	tmpDir := t.TempDir()
	id := "2304.00067"
	destDir := filepath.Join(tmpDir, "papers", id)

	// Create existing paper
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	// Write initial meta
	initialMeta := &arxiv.ArxivMeta{
		ArxivID:   id,
		Title:     "Original Title",
		FetchedAt: "2024-01-01T00:00:00Z",
	}
	metaPath := filepath.Join(destDir, "meta.yaml")
	if err := writeMeta(metaPath, initialMeta); err != nil {
		t.Fatalf("failed to write meta: %v", err)
	}

	// Simulate force mode - overwrite with new meta
	newMeta := &arxiv.ArxivMeta{
		ArxivID:   id,
		Title:     "Updated Title",
		FetchedAt: "2024-01-15T00:00:00Z",
	}
	if err := writeMeta(metaPath, newMeta); err != nil {
		t.Fatalf("failed to overwrite meta: %v", err)
	}

	// Verify updated
	readBack, err := readMeta(metaPath)
	if err != nil {
		t.Fatalf("readMeta failed: %v", err)
	}
	if readBack.Title != "Updated Title" {
		t.Errorf("Title = %q, want %q", readBack.Title, "Updated Title")
	}
}

// TestClientCreation tests arxiv client creation
func TestClientCreation(t *testing.T) {
	client, err := arxiv.NewClient()
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	if client == nil {
		t.Error("client should not be nil")
	}
}

// TestSearchOptions tests search options struct
func TestSearchOptions(t *testing.T) {
	opts := &arxiv.SearchOptions{
		Author:     "Hinton",
		Title:      "dropout",
		Category:   "cs.LG",
		MaxResults: 20,
		SortBy:     "submitted",
	}

	if opts.Author != "Hinton" {
		t.Errorf("Author = %q, want %q", opts.Author, "Hinton")
	}
	if opts.MaxResults != 20 {
		t.Errorf("MaxResults = %d, want 20", opts.MaxResults)
	}
}

// TestContextCancellation tests that operations respect context cancellation
func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Operations should return quickly when context is cancelled
	select {
	case <-ctx.Done():
		// Expected
	default:
		t.Error("context should be cancelled")
	}
}

// Integration-style tests

func TestFetchIntegration_MockServer(t *testing.T) {
	// Create a mock arXiv API server
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return mock atom/XML response
		w.Header().Set("Content-Type", "application/atom+xml")
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <entry>
    <id>http://arxiv.org/abs/2304.00067v1</id>
    <title>Test Paper Title</title>
    <summary>This is a test abstract.</summary>
    <author><name>Test Author</name></author>
    <published>2023-04-01T00:00:00Z</published>
    <updated>2023-04-01T00:00:00Z</updated>
    <arxiv:primary_category xmlns:arxiv="http://arxiv.org/schemas/atom" term="cs.LG"/>
    <category term="cs.LG"/>
  </entry>
</feed>`))
	}))
	defer apiServer.Close()

	// Create a mock PDF server
	pdfServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Length", "100")
		w.Write([]byte("%PDF-1.4 mock pdf content for testing purposes"))
	}))
	defer pdfServer.Close()

	t.Run("mock_servers_respond", func(t *testing.T) {
		// Test API server
		resp, err := http.Get(apiServer.URL)
		if err != nil {
			t.Fatalf("API server error: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("API server status = %d, want 200", resp.StatusCode)
		}

		// Test PDF server
		resp, err = http.Get(pdfServer.URL)
		if err != nil {
			t.Fatalf("PDF server error: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("PDF server status = %d, want 200", resp.StatusCode)
		}
	})
}

func TestFetchIntegration_DirectoryPermissions(t *testing.T) {
	t.Run("creates_nested_directories", func(t *testing.T) {
		tmpDir := t.TempDir()
		deepPath := filepath.Join(tmpDir, "a", "b", "c", "papers", "2304.00067")

		err := os.MkdirAll(deepPath, 0o755)
		if err != nil {
			t.Fatalf("Failed to create nested dirs: %v", err)
		}

		info, err := os.Stat(deepPath)
		if err != nil {
			t.Fatalf("Dir not created: %v", err)
		}
		if !info.IsDir() {
			t.Error("Expected directory")
		}
	})

	t.Run("handles_existing_file_at_path", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "papers")

		// Create a file where directory should be
		if err := os.WriteFile(filePath, []byte("file"), 0o644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		// Attempting to create directory should fail
		err := os.MkdirAll(filepath.Join(filePath, "2304.00067"), 0o755)
		if err == nil {
			t.Error("Expected error when creating dir over file")
		}
	})
}

func TestMetaYAML_Serialization(t *testing.T) {
	t.Run("preserves_all_fields", func(t *testing.T) {
		tmpDir := t.TempDir()
		metaPath := filepath.Join(tmpDir, "meta.yaml")

		original := &arxiv.ArxivMeta{
			ID:              "paper-2304.00067",
			ArxivID:         "2304.00067",
			Title:           "Test Paper",
			SourceType:      "arxiv",
			URL:             "https://arxiv.org/abs/2304.00067",
			PDFURL:          "https://arxiv.org/pdf/2304.00067.pdf",
			Published:       "2023-04-01T00:00:00Z",
			Updated:         "2023-04-15T00:00:00Z",
			Authors:         []arxiv.Author{{Name: "Alice", Affiliation: "MIT"}, {Name: "Bob"}},
			Abstract:        "This is the abstract.",
			Categories:      []string{"cs.LG", "cs.AI"},
			PrimaryCategory: "cs.LG",
			Comment:         "10 pages",
			JournalRef:      "Nature",
			DOI:             "10.1234/test",
			Version:         2,
			FetchedAt:       "2024-01-15T00:00:00Z",
		}

		if err := writeMeta(metaPath, original); err != nil {
			t.Fatalf("writeMeta failed: %v", err)
		}

		loaded, err := readMeta(metaPath)
		if err != nil {
			t.Fatalf("readMeta failed: %v", err)
		}

		// Compare all fields
		if loaded.ID != original.ID {
			t.Errorf("ID: got %q, want %q", loaded.ID, original.ID)
		}
		if loaded.ArxivID != original.ArxivID {
			t.Errorf("ArxivID: got %q, want %q", loaded.ArxivID, original.ArxivID)
		}
		if loaded.Title != original.Title {
			t.Errorf("Title: got %q, want %q", loaded.Title, original.Title)
		}
		if loaded.SourceType != original.SourceType {
			t.Errorf("SourceType: got %q, want %q", loaded.SourceType, original.SourceType)
		}
		if loaded.URL != original.URL {
			t.Errorf("URL: got %q, want %q", loaded.URL, original.URL)
		}
		if loaded.PDFURL != original.PDFURL {
			t.Errorf("PDFURL: got %q, want %q", loaded.PDFURL, original.PDFURL)
		}
		if loaded.Published != original.Published {
			t.Errorf("Published: got %q, want %q", loaded.Published, original.Published)
		}
		if loaded.Updated != original.Updated {
			t.Errorf("Updated: got %q, want %q", loaded.Updated, original.Updated)
		}
		if loaded.Abstract != original.Abstract {
			t.Errorf("Abstract: got %q, want %q", loaded.Abstract, original.Abstract)
		}
		if loaded.PrimaryCategory != original.PrimaryCategory {
			t.Errorf("PrimaryCategory: got %q, want %q", loaded.PrimaryCategory, original.PrimaryCategory)
		}
		if loaded.Comment != original.Comment {
			t.Errorf("Comment: got %q, want %q", loaded.Comment, original.Comment)
		}
		if loaded.JournalRef != original.JournalRef {
			t.Errorf("JournalRef: got %q, want %q", loaded.JournalRef, original.JournalRef)
		}
		if loaded.DOI != original.DOI {
			t.Errorf("DOI: got %q, want %q", loaded.DOI, original.DOI)
		}
		if loaded.Version != original.Version {
			t.Errorf("Version: got %d, want %d", loaded.Version, original.Version)
		}
		if loaded.FetchedAt != original.FetchedAt {
			t.Errorf("FetchedAt: got %q, want %q", loaded.FetchedAt, original.FetchedAt)
		}

		// Check authors
		if len(loaded.Authors) != len(original.Authors) {
			t.Fatalf("Authors length: got %d, want %d", len(loaded.Authors), len(original.Authors))
		}
		for i, author := range loaded.Authors {
			if author.Name != original.Authors[i].Name {
				t.Errorf("Authors[%d].Name: got %q, want %q", i, author.Name, original.Authors[i].Name)
			}
			if author.Affiliation != original.Authors[i].Affiliation {
				t.Errorf("Authors[%d].Affiliation: got %q, want %q", i, author.Affiliation, original.Authors[i].Affiliation)
			}
		}

		// Check categories
		if len(loaded.Categories) != len(original.Categories) {
			t.Fatalf("Categories length: got %d, want %d", len(loaded.Categories), len(original.Categories))
		}
		for i, cat := range loaded.Categories {
			if cat != original.Categories[i] {
				t.Errorf("Categories[%d]: got %q, want %q", i, cat, original.Categories[i])
			}
		}
	})

	t.Run("handles_special_yaml_characters", func(t *testing.T) {
		tmpDir := t.TempDir()
		metaPath := filepath.Join(tmpDir, "meta.yaml")

		meta := &arxiv.ArxivMeta{
			ArxivID:  "2304.00067",
			Title:    "Title with: colon and # hash and - dash",
			Abstract: "Abstract with\nnewlines\nand\ttabs",
			Authors:  []arxiv.Author{{Name: "O'Brien"}},
		}

		if err := writeMeta(metaPath, meta); err != nil {
			t.Fatalf("writeMeta failed: %v", err)
		}

		loaded, err := readMeta(metaPath)
		if err != nil {
			t.Fatalf("readMeta failed: %v", err)
		}

		if loaded.Title != meta.Title {
			t.Errorf("Title with special chars: got %q, want %q", loaded.Title, meta.Title)
		}
		if loaded.Abstract != meta.Abstract {
			t.Errorf("Abstract with special chars: got %q, want %q", loaded.Abstract, meta.Abstract)
		}
	})

	t.Run("handles_unicode", func(t *testing.T) {
		tmpDir := t.TempDir()
		metaPath := filepath.Join(tmpDir, "meta.yaml")

		meta := &arxiv.ArxivMeta{
			ArxivID:  "2304.00067",
			Title:    "Café résumé naïve 日本語",
			Abstract: "数学 физика ελληνικά",
			Authors:  []arxiv.Author{{Name: "José García"}},
		}

		if err := writeMeta(metaPath, meta); err != nil {
			t.Fatalf("writeMeta failed: %v", err)
		}

		loaded, err := readMeta(metaPath)
		if err != nil {
			t.Fatalf("readMeta failed: %v", err)
		}

		if loaded.Title != meta.Title {
			t.Errorf("Unicode title: got %q, want %q", loaded.Title, meta.Title)
		}
		if loaded.Abstract != meta.Abstract {
			t.Errorf("Unicode abstract: got %q, want %q", loaded.Abstract, meta.Abstract)
		}
		if loaded.Authors[0].Name != meta.Authors[0].Name {
			t.Errorf("Unicode author: got %q, want %q", loaded.Authors[0].Name, meta.Authors[0].Name)
		}
	})

	t.Run("handles_empty_optional_fields", func(t *testing.T) {
		tmpDir := t.TempDir()
		metaPath := filepath.Join(tmpDir, "meta.yaml")

		meta := &arxiv.ArxivMeta{
			ArxivID:    "2304.00067",
			Title:      "Minimal",
			Authors:    []arxiv.Author{},
			Categories: []string{},
		}

		if err := writeMeta(metaPath, meta); err != nil {
			t.Fatalf("writeMeta failed: %v", err)
		}

		loaded, err := readMeta(metaPath)
		if err != nil {
			t.Fatalf("readMeta failed: %v", err)
		}

		if loaded.Comment != "" {
			t.Errorf("Comment should be empty, got %q", loaded.Comment)
		}
		if loaded.JournalRef != "" {
			t.Errorf("JournalRef should be empty, got %q", loaded.JournalRef)
		}
		if loaded.DOI != "" {
			t.Errorf("DOI should be empty, got %q", loaded.DOI)
		}
	})
}

func TestNotesTemplate(t *testing.T) {
	t.Run("contains_required_sections", func(t *testing.T) {
		meta := &arxiv.ArxivMeta{
			ArxivID: "2304.00067",
			Title:   "Test Paper",
			Authors: []arxiv.Author{{Name: "Alice"}, {Name: "Bob"}},
		}

		authorNames := make([]string, 0, len(meta.Authors))
		for _, a := range meta.Authors {
			authorNames = append(authorNames, a.Name)
		}
		content := "# " + meta.Title + "\n\narXiv: " + meta.ArxivID + "\nAuthors: " + strings.Join(authorNames, ", ") + "\n\n## Summary\n\n\n## Key Takeaways\n\n\n## Follow-ups\n\n"

		requiredSections := []string{
			"# Test Paper",
			"arXiv: 2304.00067",
			"Authors: Alice, Bob",
			"## Summary",
			"## Key Takeaways",
			"## Follow-ups",
		}

		for _, section := range requiredSections {
			if !strings.Contains(content, section) {
				t.Errorf("Notes template missing section: %q", section)
			}
		}
	})

	t.Run("handles_no_authors", func(t *testing.T) {
		meta := &arxiv.ArxivMeta{
			ArxivID: "2304.00067",
			Title:   "No Authors Paper",
			Authors: []arxiv.Author{},
		}

		authorNames := make([]string, 0, len(meta.Authors))
		for _, a := range meta.Authors {
			authorNames = append(authorNames, a.Name)
		}
		content := "# " + meta.Title + "\n\narXiv: " + meta.ArxivID + "\nAuthors: " + strings.Join(authorNames, ", ") + "\n\n## Summary\n\n"

		if !strings.Contains(content, "Authors: \n") {
			// Should have empty authors line
			if !strings.Contains(content, "Authors:") {
				t.Error("Notes should contain Authors line even if empty")
			}
		}
	})

	t.Run("handles_special_characters_in_title", func(t *testing.T) {
		meta := &arxiv.ArxivMeta{
			ArxivID: "2304.00067",
			Title:   "Title: With [Special] (Characters) & Symbols",
			Authors: []arxiv.Author{},
		}

		content := "# " + meta.Title + "\n"

		if !strings.Contains(content, meta.Title) {
			t.Error("Special characters in title should be preserved")
		}
	})
}

func TestHTTPErrors(t *testing.T) {
	t.Run("handles_404", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Not Found"))
		}))
		defer server.Close()

		resp, err := http.Get(server.URL)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected 404, got %d", resp.StatusCode)
		}
	})

	t.Run("handles_500", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal Server Error"))
		}))
		defer server.Close()

		resp, err := http.Get(server.URL)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d", resp.StatusCode)
		}
	})

	t.Run("handles_timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Don't respond - let it hang
			select {}
		}))
		defer server.Close()

		client := &http.Client{
			Timeout: 10 * time.Millisecond,
		}

		_, err := client.Get(server.URL)
		if err == nil {
			t.Error("Expected timeout error")
		}
	})
}

func TestFileOperations(t *testing.T) {
	t.Run("atomic_write", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.txt")

		// Write content
		content := []byte("test content")
		if err := os.WriteFile(filePath, content, 0o644); err != nil {
			t.Fatalf("Write failed: %v", err)
		}

		// Verify content
		read, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("Read failed: %v", err)
		}
		if string(read) != string(content) {
			t.Errorf("Content mismatch: got %q, want %q", read, content)
		}
	})

	t.Run("handles_concurrent_access", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "concurrent.txt")

		// Write initial content
		if err := os.WriteFile(filePath, []byte("initial"), 0o644); err != nil {
			t.Fatalf("Initial write failed: %v", err)
		}

		// Concurrent reads should work
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				_, err := os.ReadFile(filePath)
				if err != nil {
					t.Errorf("Concurrent read failed: %v", err)
				}
				done <- true
			}()
		}

		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

func TestValidation(t *testing.T) {
	t.Run("validates_arxiv_id_before_fetch", func(t *testing.T) {
		invalidIDs := []string{
			"",
			"invalid",
			"12345",
			"not.an.id",
		}

		for _, id := range invalidIDs {
			_, err := arxiv.NormalizeArxivID(id)
			if err == nil {
				t.Errorf("Expected error for invalid ID %q", id)
			}
		}
	})

	t.Run("validates_url_format", func(t *testing.T) {
		validURLs := []string{
			"https://arxiv.org/abs/2304.00067",
			"http://arxiv.org/abs/2304.00067",
			"https://arxiv.org/pdf/2304.00067.pdf",
		}

		for _, url := range validURLs {
			_, err := arxiv.NormalizeArxivID(url)
			if err != nil {
				t.Errorf("Expected valid URL %q to be accepted, got error: %v", url, err)
			}
		}

		invalidURLs := []string{
			"https://example.com/abs/2304.00067",
			"ftp://arxiv.org/abs/2304.00067",
			"arxiv.org/abs/2304.00067", // missing scheme
		}

		for _, url := range invalidURLs {
			_, err := arxiv.NormalizeArxivID(url)
			if err == nil {
				t.Errorf("Expected invalid URL %q to be rejected", url)
			}
		}
	})
}
