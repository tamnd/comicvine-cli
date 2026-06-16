package cli

import (
	"strconv"

	"github.com/spf13/cobra"
)

func (a *App) personCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "person <name-or-id>",
		Short: "Look up a creator (writer, artist) by name or ComicVine ID",
		Long: `Look up a comic creator (writer, artist, editor, etc.) by name or numeric ID.

Examples:
  comicvine person "Stan Lee"
  comicvine person "Frank Miller"
  comicvine person 1457 -f json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ref := args[0]
			if id, err := strconv.Atoi(ref); err == nil {
				a.progressf("fetching person %d...", id)
				p, err := a.client.Person(cmd.Context(), id)
				if err != nil {
					return mapFetchErr(err)
				}
				return a.render(p)
			}
			a.progressf("searching for person %q...", ref)
			p, err := a.client.PersonByName(cmd.Context(), ref)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.render(p)
		},
	}
	return cmd
}
