package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

func (a *App) issueCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issue <id>",
		Short: "Fetch a single issue by ComicVine ID",
		Long: `Fetch a comic issue by its ComicVine numeric ID.

Examples:
  comicvine issue 100
  comicvine issue 796691 -f json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return codeError(exitUsage, fmt.Errorf("issue id must be a number, got %q", args[0]))
			}
			a.progressf("fetching issue %d...", id)
			iss, err := a.client.Issue(cmd.Context(), id)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.render(iss)
		},
	}
	return cmd
}
