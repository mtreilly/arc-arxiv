// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mtreilly/goarxiv"
	"github.com/spf13/cobra"
	"github.com/yourorg/arc-arxiv/internal/arxiv"
	"github.com/yourorg/arc-sdk/config"
)

func newExportCmd(cfg *config.Config) *cobra.Command {
	var format string
	var all bool
	var outputFile string

	cmd := &cobra.Command{
		Use:   "export [id...]",
		Short: "Export papers to BibTeX, CSV, or JSON",
		Long: `Export paper metadata in various formats.

Examples:
  arc-arxiv export 2301.12345 --format bibtex    # Single paper BibTeX
  arc-arxiv export --all --format bibtex         # All papers BibTeX
  arc-arxiv export --all --format csv            # CSV export
  arc-arxiv export --all --format json           # JSON export
  arc-arxiv export --all -f bibtex -o refs.bib   # Save to file

Formats: bibtex (default), csv, json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			papersRoot := filepath.Join(cfg.ResearchRoot, "papers")

			var metas []*arxiv.ArxivMeta

			if all {
				// Export all papers
				entries, err := os.ReadDir(papersRoot)
				if err != nil {
					if os.IsNotExist(err) {
						return fmt.Errorf("no papers found")
					}
					return err
				}

				for _, entry := range entries {
					if !entry.IsDir() {
						continue
					}
					metaPath := filepath.Join(papersRoot, entry.Name(), "meta.yaml")
					meta, err := readMeta(metaPath)
					if err != nil {
						continue
					}
					metas = append(metas, meta)
				}
			} else {
				// Export specific papers
				if len(args) == 0 {
					return fmt.Errorf("specify paper IDs or use --all to export all papers")
				}

				for _, arg := range args {
					id, err := arxiv.NormalizeArxivID(arg)
					if err != nil {
						id = arg
					}
					metaPath := filepath.Join(papersRoot, id, "meta.yaml")
					meta, err := readMeta(metaPath)
					if err != nil {
						return fmt.Errorf("paper not found: %s", id)
					}
					metas = append(metas, meta)
				}
			}

			if len(metas) == 0 {
				return fmt.Errorf("no papers to export")
			}

			var output string
			var err error

			switch strings.ToLower(format) {
			case "bibtex", "bib":
				output = exportBibTeX(metas)
			case "csv":
				output, err = exportCSV(metas)
			case "json":
				output, err = exportJSON(metas)
			default:
				return fmt.Errorf("unknown format: %s (use bibtex, csv, or json)", format)
			}

			if err != nil {
				return fmt.Errorf("export failed: %w", err)
			}

			if outputFile != "" {
				if err := os.WriteFile(outputFile, []byte(output), 0o644); err != nil {
					return fmt.Errorf("write file: %w", err)
				}
				fmt.Printf("Exported %d paper(s) to %s\n", len(metas), outputFile)
			} else {
				fmt.Print(output)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "bibtex", "Export format: bibtex, csv, json")
	cmd.Flags().BoolVar(&all, "all", false, "Export all downloaded papers")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Write output to file")

	return cmd
}

func exportBibTeX(metas []*arxiv.ArxivMeta) string {
	var entries []string
	for _, meta := range metas {
		article := arxiv.MetaToArticle(meta)
		if article != nil {
			entries = append(entries, article.ToBibTeX())
		}
	}
	return strings.Join(entries, "\n\n")
}

func exportCSV(metas []*arxiv.ArxivMeta) (string, error) {
	articles := make([]*goarxiv.Article, 0, len(metas))
	for _, meta := range metas {
		article := arxiv.MetaToArticle(meta)
		if article != nil {
			articles = append(articles, article)
		}
	}

	data, err := goarxiv.ArticlesToCSV(articles)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func exportJSON(metas []*arxiv.ArxivMeta) (string, error) {
	data, err := json.MarshalIndent(metas, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
