package cli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	var short bool
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		RunE: func(_ *cobra.Command, _ []string) error {
			if short {
				fmt.Println(Version)
				return nil
			}
			fmt.Printf("comicvine %s (commit %s, built %s, %s/%s, %s)\n",
				Version, Commit, Date, runtime.GOOS, runtime.GOARCH, runtime.Version())
			return nil
		},
	}
	cmd.Flags().BoolVar(&short, "short", false, "print just the version number")
	return cmd
}
