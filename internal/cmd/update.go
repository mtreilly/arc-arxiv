// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/mtreilly/arc-arxiv/internal/arxiv"
	"github.com/yourorg/arc-sdk/config"
)

func newUpdateCmd(cfg *config.Config) *cobra.Command {
	var all bool
	var checkOnly bool

	cmd := &cobra.Command{
		Use:   "update [id...]",
		Short: "Update paper metadata",
		Long: `Refresh metadata for downloaded papers from arXiv.

Examples:
  arc-arxiv update 2301.12345    # Update one paper
  arc-arxiv update --all         # Update all papers
  arc-arxiv update --check       # Check for new versions only

This will re-fetch metadata from arXiv and update the local meta.yaml file.
Use --check to see if newer versions are available without updating.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			papersRoot := filepath.Join(cfg.ResearchRoot, "papers")

			var ids []string

			if all {
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
					if _, err := os.Stat(metaPath); err == nil {
						ids = append(ids, entry.Name())
					}
				}
			} else {
				if len(args) == 0 {
					return fmt.Errorf("specify paper IDs or use --all to update all papers")
				}
				for _, arg := range args {
					id, err := arxiv.NormalizeArxivID(arg)
					if err != nil {
						id = arg
					}
					ids = append(ids, id)
				}
			}

			if len(ids) == 0 {
				return fmt.Errorf("no papers to update")
			}

			client, err := arxiv.NewClient()
			if err != nil {
				return fmt.Errorf("create arxiv client: %w", err)
			}

			updatedCount := 0
			newVersionCount := 0

			for _, id := range ids {
				paperDir := filepath.Join(papersRoot, id)
				metaPath := filepath.Join(paperDir, "meta.yaml")

				// Read current metadata
				currentMeta, err := readMeta(metaPath)
				if err != nil {
					fmt.Printf("  %s: not found locally, skipping\n", id)
					continue
				}

				// Fetch fresh metadata
				fmt.Printf("Checking %s...\n", id)
				newMeta, err := client.FetchArticle(ctx, id)
				if err != nil {
					fmt.Printf("  %s: failed to fetch: %v\n", id, err)
					continue
				}

				// Check for version changes
				if newMeta.Version > currentMeta.Version {
					newVersionCount++
					fmt.Printf("  %s: new version available (v%d -> v%d)\n", id, currentMeta.Version, newMeta.Version)
				}

				if checkOnly {
					continue
				}

				// Preserve fetched_at from original
				newMeta.FetchedAt = currentMeta.FetchedAt

				// Write updated metadata
				if err := writeMeta(metaPath, newMeta); err != nil {
					fmt.Printf("  %s: failed to write: %v\n", id, err)
					continue
				}

				updatedCount++
				fmt.Printf("  %s: updated\n", id)
			}

			fmt.Println()
			if checkOnly {
				if newVersionCount > 0 {
					fmt.Printf("%d paper(s) have newer versions available.\n", newVersionCount)
				} else {
					fmt.Println("All papers are up to date.")
				}
			} else {
				fmt.Printf("Updated %d paper(s).\n", updatedCount)
				if newVersionCount > 0 {
					fmt.Printf("%d paper(s) have newer versions available.\n", newVersionCount)
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "Update all downloaded papers")
	cmd.Flags().BoolVar(&checkOnly, "check", false, "Check for new versions without updating")

	return cmd
}
