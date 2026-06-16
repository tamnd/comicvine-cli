package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) searchCmd() *cobra.Command {
	var resourceType string

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search comics, characters, volumes, and more",
		Long: `Search the ComicVine database by keyword.

Examples:
  comicvine search batman
  comicvine search "spider-man" --type character
  comicvine search watchmen --type volume -f json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]
			n := a.effectiveLimit(10)
			a.progressf("searching for %q...", query)
			results, err := a.client.Search(cmd.Context(), query, resourceType, n)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(results, len(results))
		},
	}

	cmd.Flags().StringVarP(&resourceType, "type", "t", "", "resource type: character, issue, volume, publisher, person, story_arc")
	return cmd
}
