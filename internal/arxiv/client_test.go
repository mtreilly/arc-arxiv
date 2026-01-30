// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

package arxiv

import (
	"testing"
	"time"

	"github.com/mtreilly/goarxiv"
)

func TestNormalizeArxivID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		// New-style IDs (YYMM.NNNNN)
		{
			name:  "new style ID without version",
			input: "2304.00067",
			want:  "2304.00067",
		},
		{
			name:  "new style ID with version",
			input: "2304.00067v2",
			want:  "2304.00067v2",
		},
		{
			name:  "new style ID 5 digits",
			input: "2304.12345",
			want:  "2304.12345",
		},
		{
			name:  "new style ID with whitespace",
			input: "  2304.00067  ",
			want:  "2304.00067",
		},

		// Old-style IDs (archive/YYMMNNN)
		{
			name:  "old style ID hep-th",
			input: "hep-th/9901001",
			want:  "hep-th/9901001",
		},
		{
			name:  "old style ID with version",
			input: "hep-th/9901001v1",
			want:  "hep-th/9901001v1",
		},
		{
			name:  "old style ID cond-mat",
			input: "cond-mat/0001234",
			want:  "cond-mat/0001234",
		},

		// URLs - abstract page
		{
			name:  "abs URL new style",
			input: "https://arxiv.org/abs/2304.00067",
			want:  "2304.00067",
		},
		{
			name:  "abs URL with version",
			input: "https://arxiv.org/abs/2304.00067v2",
			want:  "2304.00067v2",
		},
		{
			name:  "abs URL old style",
			input: "https://arxiv.org/abs/hep-th/9901001",
			want:  "hep-th/9901001",
		},
		{
			name:  "abs URL http",
			input: "http://arxiv.org/abs/2304.00067",
			want:  "2304.00067",
		},

		// URLs - PDF
		{
			name:  "pdf URL new style",
			input: "https://arxiv.org/pdf/2304.00067.pdf",
			want:  "2304.00067",
		},
		{
			name:  "pdf URL with version",
			input: "https://arxiv.org/pdf/2304.00067v2.pdf",
			want:  "2304.00067v2",
		},
		{
			name:  "pdf URL old style",
			input: "https://arxiv.org/pdf/hep-th/9901001.pdf",
			want:  "hep-th/9901001",
		},

		// Invalid inputs
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "random text",
			input:   "not an arxiv id",
			wantErr: true,
		},
		{
			name:    "invalid format",
			input:   "1234.56",
			wantErr: true,
		},
		{
			name:    "wrong URL domain",
			input:   "https://example.com/abs/2304.00067",
			wantErr: true,
		},
		{
			name:    "partial ID",
			input:   "2304",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeArxivID(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NormalizeArxivID(%q) expected error, got %q", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Errorf("NormalizeArxivID(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.want {
				t.Errorf("NormalizeArxivID(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsValidArxivID(t *testing.T) {
	validIDs := []string{
		"2304.00067",
		"2304.00067v2",
		"hep-th/9901001",
		"https://arxiv.org/abs/2304.00067",
		"https://arxiv.org/pdf/2304.00067.pdf",
	}

	for _, id := range validIDs {
		if !IsValidArxivID(id) {
			t.Errorf("IsValidArxivID(%q) = false, want true", id)
		}
	}

	invalidIDs := []string{
		"",
		"not-valid",
		"1234",
		"https://example.com/2304.00067",
	}

	for _, id := range invalidIDs {
		if IsValidArxivID(id) {
			t.Errorf("IsValidArxivID(%q) = true, want false", id)
		}
	}
}

func TestArticleToMeta(t *testing.T) {
	published := time.Date(2023, 4, 1, 0, 0, 0, 0, time.UTC)
	updated := time.Date(2023, 4, 15, 0, 0, 0, 0, time.UTC)
	comment := "10 pages, 5 figures"
	journalRef := "Nature 2023"
	doi := "10.1234/example"
	affiliation := "MIT"

	article := &goarxiv.Article{
		ID:      "2304.00067v2",
		Title:   "  Test Paper Title  ",
		Summary: "  This is the abstract.  ",
		Authors: []goarxiv.Author{
			{Name: "Alice Smith", Affiliation: &affiliation},
			{Name: "Bob Jones", Affiliation: nil},
		},
		Published:       published,
		Updated:         updated,
		PrimaryCategory: "cs.LG",
		Categories:      []string{"cs.LG", "cs.AI"},
		Comment:         &comment,
		JournalRef:      &journalRef,
		DOI:             &doi,
	}

	meta := articleToMeta(article)

	// Test basic fields
	if meta.ArxivID != "2304.00067" {
		t.Errorf("ArxivID = %q, want %q", meta.ArxivID, "2304.00067")
	}
	if meta.ID != "paper-2304.00067" {
		t.Errorf("ID = %q, want %q", meta.ID, "paper-2304.00067")
	}
	if meta.Title != "Test Paper Title" {
		t.Errorf("Title = %q, want %q (trimmed)", meta.Title, "Test Paper Title")
	}
	if meta.Abstract != "This is the abstract." {
		t.Errorf("Abstract = %q, want %q (trimmed)", meta.Abstract, "This is the abstract.")
	}
	if meta.SourceType != "arxiv" {
		t.Errorf("SourceType = %q, want %q", meta.SourceType, "arxiv")
	}

	// Test authors
	if len(meta.Authors) != 2 {
		t.Fatalf("len(Authors) = %d, want 2", len(meta.Authors))
	}
	if meta.Authors[0].Name != "Alice Smith" {
		t.Errorf("Authors[0].Name = %q, want %q", meta.Authors[0].Name, "Alice Smith")
	}
	if meta.Authors[0].Affiliation != "MIT" {
		t.Errorf("Authors[0].Affiliation = %q, want %q", meta.Authors[0].Affiliation, "MIT")
	}
	if meta.Authors[1].Affiliation != "" {
		t.Errorf("Authors[1].Affiliation = %q, want empty", meta.Authors[1].Affiliation)
	}

	// Test categories
	if meta.PrimaryCategory != "cs.LG" {
		t.Errorf("PrimaryCategory = %q, want %q", meta.PrimaryCategory, "cs.LG")
	}
	if len(meta.Categories) != 2 {
		t.Errorf("len(Categories) = %d, want 2", len(meta.Categories))
	}

	// Test optional fields
	if meta.Comment != comment {
		t.Errorf("Comment = %q, want %q", meta.Comment, comment)
	}
	if meta.JournalRef != journalRef {
		t.Errorf("JournalRef = %q, want %q", meta.JournalRef, journalRef)
	}
	if meta.DOI != doi {
		t.Errorf("DOI = %q, want %q", meta.DOI, doi)
	}

	// Test version extraction
	if meta.Version != 2 {
		t.Errorf("Version = %d, want 2", meta.Version)
	}

	// Test URLs
	if meta.URL != "https://arxiv.org/abs/2304.00067v2" {
		t.Errorf("URL = %q, want %q", meta.URL, "https://arxiv.org/abs/2304.00067v2")
	}
	if meta.PDFURL != "https://arxiv.org/pdf/2304.00067.pdf" {
		t.Errorf("PDFURL = %q, want %q", meta.PDFURL, "https://arxiv.org/pdf/2304.00067.pdf")
	}

	// Test timestamps
	if meta.Published != published.Format(time.RFC3339) {
		t.Errorf("Published = %q, want %q", meta.Published, published.Format(time.RFC3339))
	}
	if meta.Updated != updated.Format(time.RFC3339) {
		t.Errorf("Updated = %q, want %q", meta.Updated, updated.Format(time.RFC3339))
	}

	// Test FetchedAt is set
	if meta.FetchedAt == "" {
		t.Error("FetchedAt should be set")
	}
}

func TestArticleToMeta_NilArticle(t *testing.T) {
	meta := articleToMeta(nil)
	if meta != nil {
		t.Error("articleToMeta(nil) should return nil")
	}
}

func TestArticleToMeta_EmptyOptionalFields(t *testing.T) {
	article := &goarxiv.Article{
		ID:              "2304.00067",
		Title:           "Test",
		Summary:         "Abstract",
		Authors:         []goarxiv.Author{},
		Published:       time.Now(),
		Updated:         time.Now(),
		PrimaryCategory: "cs.LG",
		Categories:      []string{},
		Comment:         nil,
		JournalRef:      nil,
		DOI:             nil,
	}

	meta := articleToMeta(article)

	if meta.Comment != "" {
		t.Errorf("Comment = %q, want empty", meta.Comment)
	}
	if meta.JournalRef != "" {
		t.Errorf("JournalRef = %q, want empty", meta.JournalRef)
	}
	if meta.DOI != "" {
		t.Errorf("DOI = %q, want empty", meta.DOI)
	}
}

func TestMetaToArticle(t *testing.T) {
	meta := &ArxivMeta{
		ID:              "paper-2304.00067",
		ArxivID:         "2304.00067",
		Title:           "Test Paper",
		Abstract:        "This is abstract",
		Authors:         []Author{{Name: "Alice", Affiliation: "MIT"}, {Name: "Bob"}},
		Published:       "2023-04-01T00:00:00Z",
		Updated:         "2023-04-15T00:00:00Z",
		PrimaryCategory: "cs.LG",
		Categories:      []string{"cs.LG", "cs.AI"},
		Comment:         "10 pages",
		JournalRef:      "Nature",
		DOI:             "10.1234/test",
	}

	article := MetaToArticle(meta)

	if article.ID != "2304.00067" {
		t.Errorf("ID = %q, want %q", article.ID, "2304.00067")
	}
	if article.Title != "Test Paper" {
		t.Errorf("Title = %q, want %q", article.Title, "Test Paper")
	}
	if article.Summary != "This is abstract" {
		t.Errorf("Summary = %q, want %q", article.Summary, "This is abstract")
	}

	// Test authors
	if len(article.Authors) != 2 {
		t.Fatalf("len(Authors) = %d, want 2", len(article.Authors))
	}
	if article.Authors[0].Name != "Alice" {
		t.Errorf("Authors[0].Name = %q, want %q", article.Authors[0].Name, "Alice")
	}
	if article.Authors[0].Affiliation == nil || *article.Authors[0].Affiliation != "MIT" {
		t.Errorf("Authors[0].Affiliation should be MIT")
	}
	if article.Authors[1].Affiliation != nil {
		t.Errorf("Authors[1].Affiliation should be nil")
	}

	// Test optional fields
	if article.Comment == nil || *article.Comment != "10 pages" {
		t.Errorf("Comment should be '10 pages'")
	}
	if article.JournalRef == nil || *article.JournalRef != "Nature" {
		t.Errorf("JournalRef should be 'Nature'")
	}
	if article.DOI == nil || *article.DOI != "10.1234/test" {
		t.Errorf("DOI should be '10.1234/test'")
	}
}

func TestMetaToArticle_NilMeta(t *testing.T) {
	article := MetaToArticle(nil)
	if article != nil {
		t.Error("MetaToArticle(nil) should return nil")
	}
}

func TestMetaToArticle_EmptyOptionalFields(t *testing.T) {
	meta := &ArxivMeta{
		ArxivID: "2304.00067",
		Title:   "Test",
		Authors: []Author{},
	}

	article := MetaToArticle(meta)

	if article.Comment != nil {
		t.Errorf("Comment should be nil")
	}
	if article.JournalRef != nil {
		t.Errorf("JournalRef should be nil")
	}
	if article.DOI != nil {
		t.Errorf("DOI should be nil")
	}
}

func TestMetaToArticle_RoundTrip(t *testing.T) {
	// Create an article, convert to meta, convert back to article
	original := &goarxiv.Article{
		ID:              "2304.00067v2",
		Title:           "Round Trip Test",
		Summary:         "Testing round trip conversion",
		Authors:         []goarxiv.Author{{Name: "Test Author"}},
		Published:       time.Date(2023, 4, 1, 0, 0, 0, 0, time.UTC),
		Updated:         time.Date(2023, 4, 15, 0, 0, 0, 0, time.UTC),
		PrimaryCategory: "cs.LG",
		Categories:      []string{"cs.LG"},
	}

	meta := articleToMeta(original)
	restored := MetaToArticle(meta)

	if restored.Title != original.Title {
		t.Errorf("Title mismatch: %q != %q", restored.Title, original.Title)
	}
	if restored.Summary != original.Summary {
		t.Errorf("Summary mismatch: %q != %q", restored.Summary, original.Summary)
	}
	if restored.PrimaryCategory != original.PrimaryCategory {
		t.Errorf("PrimaryCategory mismatch: %q != %q", restored.PrimaryCategory, original.PrimaryCategory)
	}
}

// Additional edge case tests

func TestNormalizeArxivID_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		// Version edge cases
		{
			name:  "high version number",
			input: "2304.00067v99",
			want:  "2304.00067v99",
		},
		{
			name:  "version 1 explicit",
			input: "2304.00067v1",
			want:  "2304.00067v1",
		},

		// URL variations
		{
			name:  "URL with www prefix",
			input: "https://www.arxiv.org/abs/2304.00067",
			want:  "2304.00067",
		},
		{
			name:  "URL with trailing slash",
			input: "https://arxiv.org/abs/2304.00067/",
			want:  "2304.00067",
		},
		{
			name:  "export subdomain URL",
			input: "https://export.arxiv.org/abs/2304.00067",
			want:  "2304.00067",
		},

		// Old style archive names
		{
			name:  "old style astro-ph",
			input: "astro-ph/0001001",
			want:  "astro-ph/0001001",
		},
		{
			name:  "old style quant-ph",
			input: "quant-ph/0001001",
			want:  "quant-ph/0001001",
		},
		{
			name:  "old style math",
			input: "math/0001001",
			want:  "math/0001001",
		},
		{
			name:  "old style cs",
			input: "cs/0001001",
			want:  "cs/0001001",
		},

		// Boundary cases for new IDs
		{
			name:  "minimum 4 digit paper number",
			input: "2304.0001",
			want:  "2304.0001",
		},
		{
			name:  "maximum 5 digit paper number",
			input: "2304.99999",
			want:  "2304.99999",
		},

		// Invalid edge cases
		{
			name:    "6 digit paper number",
			input:   "2304.123456",
			wantErr: true,
		},
		{
			name:    "3 digit paper number",
			input:   "2304.123",
			wantErr: true,
		},
		{
			name:    "version without number",
			input:   "2304.00067v",
			wantErr: true,
		},
		{
			name:    "negative version",
			input:   "2304.00067v-1",
			wantErr: true,
		},
		{
			name:    "only whitespace",
			input:   "   ",
			wantErr: true,
		},
		{
			name:    "newlines and tabs",
			input:   "\n\t2304.00067\n\t",
			want:    "2304.00067",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeArxivID(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NormalizeArxivID(%q) expected error, got %q", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Errorf("NormalizeArxivID(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.want {
				t.Errorf("NormalizeArxivID(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestArticleToMeta_SpecialCharacters(t *testing.T) {
	article := &goarxiv.Article{
		ID:              "2304.00067",
		Title:           "Testing: Special Characters & Symbols <in> \"Title\"",
		Summary:         "Abstract with unicode: café, naïve, 日本語",
		Authors:         []goarxiv.Author{{Name: "José García-López"}},
		Published:       time.Now(),
		Updated:         time.Now(),
		PrimaryCategory: "cs.LG",
		Categories:      []string{"cs.LG"},
	}

	meta := articleToMeta(article)

	if meta.Title != "Testing: Special Characters & Symbols <in> \"Title\"" {
		t.Errorf("Title with special chars not preserved: %q", meta.Title)
	}
	if meta.Abstract != "Abstract with unicode: café, naïve, 日本語" {
		t.Errorf("Abstract with unicode not preserved: %q", meta.Abstract)
	}
	if meta.Authors[0].Name != "José García-López" {
		t.Errorf("Author name with accents not preserved: %q", meta.Authors[0].Name)
	}
}

func TestArticleToMeta_VeryLongFields(t *testing.T) {
	// Create a very long title (1000 chars) - note: ends with space which gets trimmed
	longTitle := ""
	for i := 0; i < 100; i++ {
		longTitle += "LongTitle "
	}
	// Expected length after TrimSpace
	expectedTitleLen := len(longTitle) - 1 // trailing space removed

	// Create a very long abstract (10000 chars) - note: ends with space which gets trimmed
	longAbstract := ""
	for i := 0; i < 1000; i++ {
		longAbstract += "Abstract. "
	}
	expectedAbstractLen := len(longAbstract) - 1 // trailing space removed

	article := &goarxiv.Article{
		ID:              "2304.00067",
		Title:           longTitle,
		Summary:         longAbstract,
		Authors:         []goarxiv.Author{},
		Published:       time.Now(),
		Updated:         time.Now(),
		PrimaryCategory: "cs.LG",
		Categories:      []string{},
	}

	meta := articleToMeta(article)

	// articleToMeta calls strings.TrimSpace, so trailing space is removed
	if len(meta.Title) != expectedTitleLen {
		t.Errorf("Long title: got %d chars, want %d", len(meta.Title), expectedTitleLen)
	}
	if len(meta.Abstract) != expectedAbstractLen {
		t.Errorf("Long abstract: got %d chars, want %d", len(meta.Abstract), expectedAbstractLen)
	}

	// Also verify content is preserved (not truncated)
	if len(meta.Title) < 900 {
		t.Errorf("Title appears truncated: only %d chars", len(meta.Title))
	}
	if len(meta.Abstract) < 9000 {
		t.Errorf("Abstract appears truncated: only %d chars", len(meta.Abstract))
	}
}

func TestArticleToMeta_ManyAuthors(t *testing.T) {
	// Create 100 authors
	authors := make([]goarxiv.Author, 100)
	for i := 0; i < 100; i++ {
		authors[i] = goarxiv.Author{Name: "Author " + string(rune('A'+i%26))}
	}

	article := &goarxiv.Article{
		ID:              "2304.00067",
		Title:           "Paper with Many Authors",
		Summary:         "Abstract",
		Authors:         authors,
		Published:       time.Now(),
		Updated:         time.Now(),
		PrimaryCategory: "cs.LG",
		Categories:      []string{},
	}

	meta := articleToMeta(article)

	if len(meta.Authors) != 100 {
		t.Errorf("Expected 100 authors, got %d", len(meta.Authors))
	}
}

func TestArticleToMeta_ManyCategories(t *testing.T) {
	categories := []string{
		"cs.LG", "cs.AI", "cs.CL", "cs.CV", "cs.NE",
		"stat.ML", "math.OC", "physics.comp-ph",
	}

	article := &goarxiv.Article{
		ID:              "2304.00067",
		Title:           "Multi-category Paper",
		Summary:         "Abstract",
		Authors:         []goarxiv.Author{},
		Published:       time.Now(),
		Updated:         time.Now(),
		PrimaryCategory: "cs.LG",
		Categories:      categories,
	}

	meta := articleToMeta(article)

	if len(meta.Categories) != len(categories) {
		t.Errorf("Expected %d categories, got %d", len(categories), len(meta.Categories))
	}
}

func TestArticleToMeta_ZeroTime(t *testing.T) {
	article := &goarxiv.Article{
		ID:              "2304.00067",
		Title:           "Test",
		Summary:         "Abstract",
		Authors:         []goarxiv.Author{},
		Published:       time.Time{}, // zero time
		Updated:         time.Time{}, // zero time
		PrimaryCategory: "cs.LG",
		Categories:      []string{},
	}

	meta := articleToMeta(article)

	// Should still produce valid RFC3339 strings
	if meta.Published == "" {
		t.Error("Published should not be empty even for zero time")
	}
	if meta.Updated == "" {
		t.Error("Updated should not be empty even for zero time")
	}
}

func TestProgressWriter(t *testing.T) {
	var calls []struct {
		downloaded int64
		total      int64
	}

	pw := &progressWriter{
		total: 1000,
		cb: func(downloaded, total int64) {
			calls = append(calls, struct {
				downloaded int64
				total      int64
			}{downloaded, total})
		},
	}

	// Write in chunks
	pw.Write([]byte("12345"))     // 5 bytes
	pw.Write([]byte("1234567890")) // 10 bytes
	pw.Write([]byte("123"))        // 3 bytes

	if len(calls) != 3 {
		t.Errorf("Expected 3 progress callbacks, got %d", len(calls))
	}

	if calls[0].downloaded != 5 {
		t.Errorf("First callback: downloaded = %d, want 5", calls[0].downloaded)
	}
	if calls[1].downloaded != 15 {
		t.Errorf("Second callback: downloaded = %d, want 15", calls[1].downloaded)
	}
	if calls[2].downloaded != 18 {
		t.Errorf("Third callback: downloaded = %d, want 18", calls[2].downloaded)
	}

	// All should report same total
	for i, call := range calls {
		if call.total != 1000 {
			t.Errorf("Callback %d: total = %d, want 1000", i, call.total)
		}
	}
}

func TestProgressWriter_NilCallback(t *testing.T) {
	pw := &progressWriter{
		total: 1000,
		cb:    nil,
	}

	// Should not panic
	n, err := pw.Write([]byte("test"))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if n != 4 {
		t.Errorf("Write returned %d, want 4", n)
	}
}

func TestProgressWriter_ZeroTotal(t *testing.T) {
	var lastDownloaded int64

	pw := &progressWriter{
		total: 0, // unknown total
		cb: func(downloaded, total int64) {
			lastDownloaded = downloaded
		},
	}

	pw.Write([]byte("test data"))

	if lastDownloaded != 9 {
		t.Errorf("downloaded = %d, want 9", lastDownloaded)
	}
}

func TestAuthor_EmptyName(t *testing.T) {
	article := &goarxiv.Article{
		ID:      "2304.00067",
		Title:   "Test",
		Summary: "Abstract",
		Authors: []goarxiv.Author{
			{Name: ""},
			{Name: "Valid Author"},
			{Name: "   "}, // whitespace only
		},
		Published:       time.Now(),
		Updated:         time.Now(),
		PrimaryCategory: "cs.LG",
		Categories:      []string{},
	}

	meta := articleToMeta(article)

	// Empty names should still be preserved (the consumer can filter)
	if len(meta.Authors) != 3 {
		t.Errorf("Expected 3 authors, got %d", len(meta.Authors))
	}
}
