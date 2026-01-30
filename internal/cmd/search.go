// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/mtreilly/arc-arxiv/internal/arxiv"
	"github.com/yourorg/arc-sdk/config"
	"github.com/yourorg/arc-sdk/output"
)

func newSearchCmd(cfg *config.Config) *cobra.Command {
	var out output.OutputOptions
	var author string
	var title string
	var abstract string
	var category string
	var maxResults int
	var sortBy string
	var fetch bool

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search arXiv papers",
		Long: `Search arXiv for papers matching the query.

Examples:
  arc-arxiv search "transformer attention"              # Free-text search
  arc-arxiv search --author "Hinton" --title "dropout"  # Field search
  arc-arxiv search --category cs.LG --max 20            # Category filter
  arc-arxiv search "neural networks" --sort submitted   # Sort by submission date
  arc-arxiv search "quantum computing" --fetch          # Auto-fetch top results

Sort options: relevance (default), submitted, updated`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := out.Resolve(); err != nil {
				return err
			}

			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			query := ""
			if len(args) > 0 {
				query = args[0]
			}

			// Require at least some search criteria
			if query == "" && author == "" && title == "" && abstract == "" && category == "" {
				return fmt.Errorf("please provide a search query or use --author, --title, --abstract, or --category flags")
			}

			client, err := arxiv.NewClient()
			if err != nil {
				return fmt.Errorf("create arxiv client: %w", err)
			}

			opts := &arxiv.SearchOptions{
				Author:     author,
				Title:      title,
				Abstract:   abstract,
				Category:   category,
				MaxResults: maxResults,
				SortBy:     sortBy,
			}

			fmt.Printf("Searching arXiv...\n")
			results, totalResults, err := client.Search(ctx, query, opts)
			if err != nil {
				return fmt.Errorf("search failed: %w", err)
			}

			if len(results) == 0 {
				fmt.Println("No results found.")
				return nil
			}

			fmt.Printf("Found %d results (showing %d)\n\n", totalResults, len(results))

			if out.Is(output.OutputJSON) {
				return output.JSON(results)
			}

			// Display results in table format
			table := output.NewTable("ID", "Title", "Authors", "Published")
			for _, r := range results {
				title := truncate(r.Title, 45)
				authors := ""
				if len(r.Authors) > 0 {
					names := make([]string, 0, len(r.Authors))
					for i, a := range r.Authors {
						if i >= 3 {
							names = append(names, "...")
							break
						}
						names = append(names, a.Name)
					}
					authors = truncate(strings.Join(names, ", "), 30)
				}
				published := ""
				if r.Published != "" {
					published = r.Published[:10] // Just the date part
				}
				table.AddRow(r.ArxivID, title, authors, published)
			}
			table.Render()

			// Auto-fetch if requested
			if fetch && len(results) > 0 {
				fmt.Printf("\nFetching top %d results...\n", len(results))
				fetchCmd := newFetchCmd(cfg)
				ids := make([]string, 0, len(results))
				for _, r := range results {
					ids = append(ids, r.ArxivID)
				}
				fetchCmd.SetContext(ctx)
				return fetchCmd.RunE(fetchCmd, ids)
			}

			return nil
		},
	}

	out.AddOutputFlags(cmd, output.OutputTable)
	cmd.Flags().StringVarP(&author, "author", "a", "", "Filter by author name")
	cmd.Flags().StringVarP(&title, "title", "t", "", "Filter by title")
	cmd.Flags().StringVar(&abstract, "abstract", "", "Filter by abstract content")
	cmd.Flags().StringVarP(&category, "category", "c", "", "Filter by category (e.g., cs.LG, physics.hep-th)")
	cmd.Flags().IntVarP(&maxResults, "max", "m", 10, "Maximum number of results")
	cmd.Flags().StringVarP(&sortBy, "sort", "s", "relevance", "Sort by: relevance, submitted, updated")
	cmd.Flags().BoolVar(&fetch, "fetch", false, "Automatically fetch all results")

	return cmd
}
