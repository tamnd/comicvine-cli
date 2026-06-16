package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

func (a *App) publisherCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "publisher <id>",
		Short: "Fetch a publisher by ComicVine ID",
		Long: `Fetch a comic publisher by its ComicVine numeric ID.

Examples:
  comicvine publisher 10
  comicvine publisher 31 -f json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return codeError(exitUsage, fmt.Errorf("publisher id must be a number, got %q", args[0]))
			}
			a.progressf("fetching publisher %d...", id)
			pub, err := a.client.Publisher(cmd.Context(), id)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.render(pub)
		},
	}
	return cmd
}
