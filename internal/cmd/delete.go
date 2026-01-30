// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/mtreilly/arc-arxiv/internal/arxiv"
	"github.com/yourorg/arc-sdk/config"
)

func newDeleteCmd(cfg *config.Config) *cobra.Command {
	var force bool
	var dryRun bool

	cmd := &cobra.Command{
		Use:     "delete <id> [id...]",
		Aliases: []string{"rm", "remove"},
		Short:   "Delete downloaded papers",
		Long: `Remove downloaded papers from the local filesystem.

Examples:
  arc-arxiv delete 2304.00067           # Delete one paper (with confirmation)
  arc-arxiv delete 2304.00067 --force   # Delete without confirmation
  arc-arxiv delete 2304.00067 --dry-run # Show what would be deleted`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			papersRoot := filepath.Join(cfg.ResearchRoot, "papers")

			// Normalize and validate all IDs first
			var toDelete []struct {
				id   string
				path string
				meta *arxiv.ArxivMeta
			}

			for _, arg := range args {
				id, err := arxiv.NormalizeArxivID(arg)
				if err != nil {
					id = arg
				}

				paperDir := filepath.Join(papersRoot, id)
				if _, err := os.Stat(paperDir); os.IsNotExist(err) {
					fmt.Printf("Paper not found: %s\n", id)
					continue
				}

				metaPath := filepath.Join(paperDir, "meta.yaml")
				meta, _ := readMeta(metaPath)

				toDelete = append(toDelete, struct {
					id   string
					path string
					meta *arxiv.ArxivMeta
				}{id: id, path: paperDir, meta: meta})
			}

			if len(toDelete) == 0 {
				return fmt.Errorf("no papers found to delete")
			}

			// Show what will be deleted
			fmt.Printf("Papers to delete:\n")
			for _, p := range toDelete {
				title := p.id
				if p.meta != nil && p.meta.Title != "" {
					title = truncate(p.meta.Title, 60)
				}
				fmt.Printf("  %s - %s\n", p.id, title)
			}
			fmt.Println()

			if dryRun {
				fmt.Printf("[dry-run] Would delete %d paper(s)\n", len(toDelete))
				return nil
			}

			// Confirm unless --force
			if !force {
				fmt.Printf("Delete %d paper(s)? [y/N] ", len(toDelete))
				reader := bufio.NewReader(os.Stdin)
				response, err := reader.ReadString('\n')
				if err != nil {
					return err
				}
				response = strings.TrimSpace(strings.ToLower(response))
				if response != "y" && response != "yes" {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			// Delete papers
			deleted := 0
			for _, p := range toDelete {
				if err := os.RemoveAll(p.path); err != nil {
					fmt.Printf("Failed to delete %s: %v\n", p.id, err)
					continue
				}
				fmt.Printf("Deleted: %s\n", p.id)
				deleted++
			}

			fmt.Printf("\nDeleted %d paper(s).\n", deleted)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Delete without confirmation")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "Show what would be deleted")

	return cmd
}
