package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

func (a *App) issuesCmd() *cobra.Command {
	var from, to int

	cmd := &cobra.Command{
		Use:   "issues <volume-id>",
		Short: "List issues within a volume",
		Long: `List issues in a volume (series), in issue number order.

Examples:
  comicvine issues 796
  comicvine issues 796 --from 1 --to 50
  comicvine issues 796 -n 100 -f jsonl`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			volID, err := strconv.Atoi(args[0])
			if err != nil {
				return codeError(exitUsage, fmt.Errorf("volume id must be a number, got %q", args[0]))
			}
			n := a.effectiveLimit(20)
			a.progressf("fetching issues for volume %d...", volID)
			issues, err := a.client.Issues(cmd.Context(), volID, from, to, n)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(issues, len(issues))
		},
	}

	cmd.Flags().IntVar(&from, "from", 0, "start issue number (inclusive, 0 = first)")
	cmd.Flags().IntVar(&to, "to", 0, "end issue number (inclusive, 0 = all)")
	return cmd
}
