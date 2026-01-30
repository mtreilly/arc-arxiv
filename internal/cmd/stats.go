// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yourorg/arc-sdk/config"
	"github.com/yourorg/arc-sdk/output"
)

func newStatsCmd(cfg *config.Config) *cobra.Command {
	var out output.OutputOptions

	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Show library statistics",
		Long: `Display statistics about downloaded papers.

Shows counts by category, author, publication year, and fetch date.`,
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

			// Collect statistics
			stats := &libraryStats{
				Categories:    make(map[string]int),
				Authors:       make(map[string]int),
				Years:         make(map[int]int),
				FetchedMonths: make(map[string]int),
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

				stats.TotalPapers++

				// Count categories
				for _, cat := range meta.Categories {
					stats.Categories[cat]++
				}

				// Count authors
				for _, author := range meta.Authors {
					stats.Authors[author.Name]++
				}

				// Count publication years
				if meta.Published != "" {
					if t, err := time.Parse(time.RFC3339, meta.Published); err == nil {
						stats.Years[t.Year()]++
					}
				}

				// Count fetch months
				if meta.FetchedAt != "" {
					if t, err := time.Parse(time.RFC3339, meta.FetchedAt); err == nil {
						month := t.Format("2006-01")
						stats.FetchedMonths[month]++
					}
				}
			}

			if stats.TotalPapers == 0 {
				fmt.Println("No papers found.")
				return nil
			}

			if out.Is(output.OutputJSON) {
				return output.JSON(stats)
			}

			// Display statistics
			fmt.Printf("Library Statistics\n")
			fmt.Printf("==================\n\n")
			fmt.Printf("Total papers: %d\n\n", stats.TotalPapers)

			// Top categories
			fmt.Printf("Categories:\n")
			topCategories := topN(stats.Categories, 10)
			for _, kv := range topCategories {
				fmt.Printf("  %-20s %d\n", kv.Key, kv.Value)
			}
			if len(stats.Categories) > 10 {
				fmt.Printf("  ... and %d more\n", len(stats.Categories)-10)
			}
			fmt.Println()

			// Top authors
			fmt.Printf("Top Authors:\n")
			topAuthors := topN(stats.Authors, 10)
			for _, kv := range topAuthors {
				name := kv.Key.(string)
				if len(name) > 30 {
					name = name[:27] + "..."
				}
				fmt.Printf("  %-30s %d\n", name, kv.Value)
			}
			if len(stats.Authors) > 10 {
				fmt.Printf("  ... and %d more\n", len(stats.Authors)-10)
			}
			fmt.Println()

			// Publication years
			fmt.Printf("Publication Years:\n")
			years := topN(stats.Years, 10)
			// Sort by year descending
			sort.Slice(years, func(i, j int) bool {
				yi, _ := years[i].Key.(int)
				yj, _ := years[j].Key.(int)
				return yi > yj
			})
			for _, kv := range years {
				fmt.Printf("  %v: %d\n", kv.Key, kv.Value)
			}
			fmt.Println()

			// Fetch activity
			fmt.Printf("Fetch Activity:\n")
			months := topN(stats.FetchedMonths, 6)
			// Sort by month descending
			sort.Slice(months, func(i, j int) bool {
				return strings.Compare(months[i].Key.(string), months[j].Key.(string)) > 0
			})
			for _, kv := range months {
				fmt.Printf("  %s: %d\n", kv.Key, kv.Value)
			}

			return nil
		},
	}

	out.AddOutputFlags(cmd, output.OutputTable)

	return cmd
}

type libraryStats struct {
	TotalPapers   int            `json:"total_papers"`
	Categories    map[string]int `json:"categories"`
	Authors       map[string]int `json:"authors"`
	Years         map[int]int    `json:"years"`
	FetchedMonths map[string]int `json:"fetched_months"`
}

type kv struct {
	Key   any
	Value int
}

func topN[K comparable](m map[K]int, n int) []kv {
	var pairs []kv
	for k, v := range m {
		pairs = append(pairs, kv{Key: k, Value: v})
	}
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Value > pairs[j].Value
	})
	if len(pairs) > n {
		pairs = pairs[:n]
	}
	return pairs
}
