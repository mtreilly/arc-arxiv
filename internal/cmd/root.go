// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/mtreilly/arc-arxiv/internal/arxiv"
	"github.com/yourorg/arc-sdk/config"
	"github.com/yourorg/arc-sdk/output"
	"github.com/yourorg/arc-sdk/utils"
	"gopkg.in/yaml.v3"
)

// NewRootCmd creates the root command for arc-arxiv.
func NewRootCmd(cfg *config.Config, db *sql.DB) *cobra.Command {
	root := &cobra.Command{
		Use:   "arc-arxiv",
		Short: "Fetch and manage arXiv papers",
		Long: `Download arXiv papers (metadata + PDF) into your research workspace.

Papers are saved to the research root under papers/<arxiv-id>/ with:
- meta.yaml: Paper metadata
- paper.pdf: The PDF file
- notes.md: Template for your notes`,
	}

	root.AddCommand(newFetchCmd(cfg))
	root.AddCommand(newListCmd(cfg))
	root.AddCommand(newInfoCmd(cfg))
	root.AddCommand(newOpenCmd(cfg))
	root.AddCommand(newSearchCmd(cfg))
	root.AddCommand(newExportCmd(cfg))
	root.AddCommand(newUpdateCmd(cfg))
	root.AddCommand(newDeleteCmd(cfg))
	root.AddCommand(newStatsCmd(cfg))

	return root
}

func newFetchCmd(cfg *config.Config) *cobra.Command {
	var extractText bool
	var openNotes bool
	var dryRun bool
	var force bool

	cmd := &cobra.Command{
		Use:   "fetch <id-or-url> [id-or-url...]",
		Short: "Fetch arXiv papers",
		Long: `Download arXiv papers (metadata + PDF) into the research workspace.

Accepts arXiv IDs or URLs in any format:
  arc-arxiv fetch 2304.00067
  arc-arxiv fetch https://arxiv.org/abs/2304.00067
  arc-arxiv fetch https://arxiv.org/pdf/2304.00067.pdf

Multiple papers can be fetched at once:
  arc-arxiv fetch 2304.00067 2301.12345 2312.99999

Each paper is saved to research_root/papers/<arxiv-id>/ with meta.yaml,
paper.pdf, and notes.md files.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			papersRoot := filepath.Join(cfg.ResearchRoot, "papers")

			// Normalize all IDs first
			ids := make([]string, 0, len(args))
			for _, input := range args {
				id, err := arxiv.NormalizeArxivID(input)
				if err != nil {
					return fmt.Errorf("invalid arXiv ID or URL: %s", input)
				}
				ids = append(ids, id)
			}

			// Create arxiv client
			client, err := arxiv.NewClient()
			if err != nil {
				return fmt.Errorf("create arxiv client: %w", err)
			}

			for _, id := range ids {
				destDir := filepath.Join(papersRoot, id)

				if _, err := os.Stat(destDir); err == nil {
					if !force {
						fmt.Printf("Paper %s already exists at %s (use --force to re-fetch)\n", id, destDir)
						continue
					}
					fmt.Printf("Re-fetching paper %s...\n", id)
				}

				if dryRun {
					fmt.Printf("[dry-run] Would fetch paper:\n")
					fmt.Printf("  ID: %s\n", id)
					fmt.Printf("  Directory: %s\n", destDir)
					fmt.Printf("  Files: paper.pdf, meta.yaml, notes.md\n")
					continue
				}

				fmt.Printf("Fetching metadata for %s...\n", id)
				meta, err := client.FetchArticle(ctx, id)
				if err != nil {
					return fmt.Errorf("fetch metadata for %s: %w", id, err)
				}

				// Create directory
				if err := os.MkdirAll(destDir, 0o755); err != nil {
					return fmt.Errorf("create directory: %w", err)
				}

				// Download PDF with progress
				pdfPath := filepath.Join(destDir, "paper.pdf")
				fmt.Printf("Downloading PDF: %s\n", meta.PDFURL)

				var lastProgress int
				err = client.DownloadPDF(ctx, id, pdfPath, func(downloaded, total int64) {
					if total > 0 {
						pct := int(float64(downloaded) / float64(total) * 100)
						if pct >= lastProgress+10 || pct == 100 {
							fmt.Printf("\r  Progress: %d%%", pct)
							lastProgress = pct
						}
					}
				})
				fmt.Println()

				if err != nil {
					_ = os.RemoveAll(destDir)
					return fmt.Errorf("download PDF: %w", err)
				}

				// Write meta.yaml
				metaPath := filepath.Join(destDir, "meta.yaml")
				if err := writeMeta(metaPath, meta); err != nil {
					return fmt.Errorf("write meta: %w", err)
				}

				// Create notes template
				notesPath := filepath.Join(destDir, "notes.md")
				authorNames := make([]string, 0, len(meta.Authors))
				for _, a := range meta.Authors {
					authorNames = append(authorNames, a.Name)
				}
				notesContent := fmt.Sprintf("# %s\n\narXiv: %s\nAuthors: %s\n\n## Summary\n\n\n## Key Takeaways\n\n\n## Follow-ups\n\n",
					meta.Title, id, strings.Join(authorNames, ", "))
				if err := os.WriteFile(notesPath, []byte(notesContent), 0o644); err != nil {
					return fmt.Errorf("write notes: %w", err)
				}

				// Extract text if requested
				if extractText {
					bodyPath := filepath.Join(destDir, "body.md")
					if err := extractPdfText(ctx, pdfPath, bodyPath); err != nil {
						fmt.Printf("Warning: text extraction failed: %v\n", err)
					}
				}

				// Print summary
				fmt.Printf("\nSaved: %s\n", destDir)
				fmt.Printf("  Title: %s\n", truncate(meta.Title, 70))
				if len(authorNames) > 0 {
					fmt.Printf("  Authors: %s\n", truncate(strings.Join(authorNames, ", "), 70))
				}
				if len(meta.Categories) > 0 {
					fmt.Printf("  Categories: %s\n", strings.Join(meta.Categories, ", "))
				}
				fmt.Println()

				if openNotes {
					_ = openFile(ctx, notesPath)
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&extractText, "extract-text", "x", false, "Extract PDF text into body.md")
	cmd.Flags().BoolVarP(&openNotes, "notes", "n", false, "Open notes.md after creation")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "Show planned actions without writing files")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Re-fetch even if paper already exists")

	return cmd
}

func newListCmd(cfg *config.Config) *cobra.Command {
	var out output.OutputOptions
	var category string
	var author string
	var since string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List downloaded papers",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := out.Resolve(); err != nil {
				return err
			}

			papersRoot := filepath.Join(cfg.ResearchRoot, "papers")
			entries, err := os.ReadDir(papersRoot)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Println("No papers downloaded yet.")
					return nil
				}
				return err
			}

			var sinceTime time.Time
			if since != "" {
				var err error
				sinceTime, err = time.Parse("2006-01-02", since)
				if err != nil {
					return fmt.Errorf("invalid date format for --since (use YYYY-MM-DD): %w", err)
				}
			}

			var papers []*arxiv.ArxivMeta
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				metaPath := filepath.Join(papersRoot, entry.Name(), "meta.yaml")
				meta, err := readMeta(metaPath)
				if err != nil {
					continue
				}

				// Apply filters
				if category != "" {
					found := false
					for _, c := range meta.Categories {
						if strings.EqualFold(c, category) || strings.HasPrefix(strings.ToLower(c), strings.ToLower(category)) {
							found = true
							break
						}
					}
					if !found {
						continue
					}
				}

				if author != "" {
					found := false
					authorLower := strings.ToLower(author)
					for _, a := range meta.Authors {
						if strings.Contains(strings.ToLower(a.Name), authorLower) {
							found = true
							break
						}
					}
					if !found {
						continue
					}
				}

				if !sinceTime.IsZero() {
					fetchedAt, err := time.Parse(time.RFC3339, meta.FetchedAt)
					if err != nil || fetchedAt.Before(sinceTime) {
						continue
					}
				}

				papers = append(papers, meta)
			}

			if len(papers) == 0 {
				fmt.Println("No papers found.")
				return nil
			}

			if out.Is(output.OutputJSON) {
				return output.JSON(papers)
			}

			table := output.NewTable("ID", "Title", "Authors", "Fetched")
			for _, p := range papers {
				title := truncate(p.Title, 40)
				authors := ""
				if len(p.Authors) > 0 {
					names := make([]string, 0, len(p.Authors))
					for _, a := range p.Authors {
						names = append(names, a.Name)
					}
					authors = truncate(strings.Join(names, ", "), 30)
				}
				table.AddRow(p.ArxivID, title, authors, utils.HumanizeTime(parseTime(p.FetchedAt)))
			}
			table.Render()

			return nil
		},
	}

	out.AddOutputFlags(cmd, output.OutputTable)
	cmd.Flags().StringVarP(&category, "category", "c", "", "Filter by category (e.g., cs.LG)")
	cmd.Flags().StringVarP(&author, "author", "a", "", "Filter by author name")
	cmd.Flags().StringVar(&since, "since", "", "Filter papers fetched after date (YYYY-MM-DD)")

	return cmd
}

func newInfoCmd(cfg *config.Config) *cobra.Command {
	var out output.OutputOptions

	cmd := &cobra.Command{
		Use:   "info <id>",
		Short: "Show paper details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := out.Resolve(); err != nil {
				return err
			}

			id, err := arxiv.NormalizeArxivID(args[0])
			if err != nil {
				id = args[0] // fallback to raw input for local lookup
			}
			metaPath := filepath.Join(cfg.ResearchRoot, "papers", id, "meta.yaml")

			meta, err := readMeta(metaPath)
			if err != nil {
				return fmt.Errorf("paper not found: %s", id)
			}

			if out.Is(output.OutputJSON) {
				return output.JSON(meta)
			}

			fmt.Printf("ID:              %s\n", meta.ArxivID)
			fmt.Printf("Title:           %s\n", meta.Title)
			fmt.Printf("URL:             %s\n", meta.URL)
			fmt.Printf("PDF:             %s\n", meta.PDFURL)
			if len(meta.Authors) > 0 {
				names := make([]string, 0, len(meta.Authors))
				for _, a := range meta.Authors {
					if a.Affiliation != "" {
						names = append(names, fmt.Sprintf("%s (%s)", a.Name, a.Affiliation))
					} else {
						names = append(names, a.Name)
					}
				}
				fmt.Printf("Authors:         %s\n", strings.Join(names, ", "))
			}
			if meta.PrimaryCategory != "" {
				fmt.Printf("Primary Category: %s\n", meta.PrimaryCategory)
			}
			if len(meta.Categories) > 0 {
				fmt.Printf("Categories:      %s\n", strings.Join(meta.Categories, ", "))
			}
			if meta.Published != "" {
				fmt.Printf("Published:       %s\n", meta.Published)
			}
			if meta.Updated != "" {
				fmt.Printf("Updated:         %s\n", meta.Updated)
			}
			if meta.Version > 0 {
				fmt.Printf("Version:         v%d\n", meta.Version)
			}
			if meta.DOI != "" {
				fmt.Printf("DOI:             %s\n", meta.DOI)
			}
			if meta.JournalRef != "" {
				fmt.Printf("Journal Ref:     %s\n", meta.JournalRef)
			}
			if meta.Comment != "" {
				fmt.Printf("Comment:         %s\n", meta.Comment)
			}
			if meta.Abstract != "" {
				fmt.Printf("\nAbstract:\n%s\n", meta.Abstract)
			}

			return nil
		},
	}

	out.AddOutputFlags(cmd, output.OutputTable)

	return cmd
}

func newOpenCmd(cfg *config.Config) *cobra.Command {
	var pdf bool
	var notes bool
	var web bool

	cmd := &cobra.Command{
		Use:   "open <id>",
		Short: "Open a paper",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := arxiv.NormalizeArxivID(args[0])
			if err != nil {
				id = args[0]
			}
			paperDir := filepath.Join(cfg.ResearchRoot, "papers", id)

			if _, err := os.Stat(paperDir); os.IsNotExist(err) {
				return fmt.Errorf("paper not found: %s", id)
			}

			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			if web {
				url := fmt.Sprintf("https://arxiv.org/abs/%s", id)
				return openURL(ctx, url)
			}

			if pdf {
				pdfPath := filepath.Join(paperDir, "paper.pdf")
				return openFile(ctx, pdfPath)
			}

			if notes {
				notesPath := filepath.Join(paperDir, "notes.md")
				return openFile(ctx, notesPath)
			}

			// Default: open the directory
			return openFile(ctx, paperDir)
		},
	}

	cmd.Flags().BoolVar(&pdf, "pdf", false, "Open the PDF file")
	cmd.Flags().BoolVar(&notes, "notes", false, "Open the notes file")
	cmd.Flags().BoolVar(&web, "web", false, "Open in browser")

	return cmd
}

// Helper functions

func writeMeta(path string, meta *arxiv.ArxivMeta) error {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(meta); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

func readMeta(path string) (*arxiv.ArxivMeta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var meta arxiv.ArxivMeta
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func extractPdfText(ctx context.Context, pdfPath, bodyPath string) error {
	if _, err := exec.LookPath("pdftotext"); err != nil {
		return fmt.Errorf("pdftotext not available")
	}
	cmd := exec.CommandContext(ctx, "pdftotext", pdfPath, bodyPath)
	return cmd.Run()
}

func openFile(ctx context.Context, path string) error {
	var cmd *exec.Cmd
	if _, err := exec.LookPath("open"); err == nil {
		cmd = exec.CommandContext(ctx, "open", path)
	} else if _, err := exec.LookPath("xdg-open"); err == nil {
		cmd = exec.CommandContext(ctx, "xdg-open", path)
	} else {
		return fmt.Errorf("no file opener available")
	}
	return cmd.Start()
}

func openURL(ctx context.Context, url string) error {
	return openFile(ctx, url)
}

func parseTime(s string) int64 {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return 0
	}
	return t.Unix()
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
