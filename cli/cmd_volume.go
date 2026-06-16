package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

func (a *App) volumeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "volume <id>",
		Short: "Fetch a volume/series by ComicVine ID",
		Long: `Fetch a comic series (volume) by its ComicVine numeric ID.

Examples:
  comicvine volume 796
  comicvine volume 796 -f json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return codeError(exitUsage, fmt.Errorf("volume id must be a number, got %q", args[0]))
			}
			a.progressf("fetching volume %d...", id)
			vol, err := a.client.Volume(cmd.Context(), id)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.render(vol)
		},
	}
	return cmd
}
