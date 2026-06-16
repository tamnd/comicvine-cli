package cli

import (
	"strconv"

	"github.com/spf13/cobra"
)

func (a *App) characterCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "character <name-or-id>",
		Short: "Look up a character by name or ComicVine ID",
		Long: `Look up a comic character by name (first match) or numeric ID.

Examples:
  comicvine character batman
  comicvine character "Jean Grey"
  comicvine character 1490 -f json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ref := args[0]
			if id, err := strconv.Atoi(ref); err == nil {
				a.progressf("fetching character %d...", id)
				ch, err := a.client.Character(cmd.Context(), id)
				if err != nil {
					return mapFetchErr(err)
				}
				return a.render(ch)
			}
			a.progressf("searching for character %q...", ref)
			ch, err := a.client.CharacterByName(cmd.Context(), ref)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.render(ch)
		},
	}
	return cmd
}
