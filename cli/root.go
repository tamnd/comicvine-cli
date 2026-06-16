// Package cli builds the comicvine command tree on top of the comicvine library.
package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"github.com/tamnd/comicvine-cli/comicvine"
)

// Build metadata, set via -ldflags at release time.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// exit codes.
const (
	exitError       = 1
	exitUsage       = 2
	exitNoData      = 3
	exitRateLimited = 5
)

// ExitError carries a process exit code up to main.
type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return fmt.Sprintf("exit %d", e.Code)
}

func (e *ExitError) Unwrap() error { return e.Err }

func codeError(code int, err error) error { return &ExitError{Code: code, Err: err} }

// App holds shared state threaded through every command.
type App struct {
	client *comicvine.Client
	cfg    comicvine.Config
	apiKey string

	output   string
	fields   []string
	noHeader bool
	template string
	limit    int
	quiet    bool
}

// Root builds the root command and its subtree.
func Root() *cobra.Command {
	app := &App{cfg: comicvine.DefaultConfig()}

	root := &cobra.Command{
		Use:   "comicvine",
		Short: "Browse the ComicVine comic book database",
		Long: `comicvine reads the public ComicVine REST API at https://comicvine.gamespot.com/api/
It browses characters, issues, volumes, publishers, and creators.
Requires COMICVINE_API_KEY (free registration at https://comicvine.gamespot.com/api/).

comicvine is an independent tool and is not affiliated with ComicVine or GameSpot.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if cmd.Name() == "version" {
				return nil
			}
			return app.setup()
		},
	}

	pf := root.PersistentFlags()
	pf.StringVarP(&app.output, "format", "f", "auto", "output format: table|json|jsonl|csv|tsv|url (auto=table on TTY, jsonl piped)")
	pf.StringSliceVar(&app.fields, "fields", nil, "comma-separated columns to include")
	pf.BoolVar(&app.noHeader, "no-header", false, "omit the header row in table/csv/tsv")
	pf.StringVar(&app.template, "template", "", "Go text/template applied per record")
	pf.IntVarP(&app.limit, "limit", "n", 0, "limit number of records (0 = command default)")
	pf.BoolVarP(&app.quiet, "quiet", "q", false, "suppress progress on stderr")

	pf.StringVar(&app.cfg.BaseURL, "base-url", app.cfg.BaseURL, "ComicVine API base URL")
	pf.DurationVar(&app.cfg.Rate, "delay", app.cfg.Rate, "minimum spacing between requests")
	pf.DurationVar(&app.cfg.Timeout, "timeout", app.cfg.Timeout, "per-request timeout")
	pf.IntVar(&app.cfg.Retries, "retries", app.cfg.Retries, "retry attempts on 429/5xx")
	pf.StringVar(&app.cfg.UserAgent, "user-agent", app.cfg.UserAgent, "User-Agent sent with each request")

	root.AddCommand(
		app.searchCmd(),
		app.issueCmd(),
		app.volumeCmd(),
		app.characterCmd(),
		app.issuesCmd(),
		app.publisherCmd(),
		app.personCmd(),
		newVersionCmd(),
	)
	return root
}

func (a *App) setup() error {
	if a.output == "" || a.output == "auto" {
		if isatty.IsTerminal(os.Stdout.Fd()) {
			a.output = string(FormatTable)
		} else {
			a.output = string(FormatJSONL)
		}
	}
	if !Format(a.output).Valid() {
		return codeError(exitUsage, fmt.Errorf("unknown output format %q", a.output))
	}

	// API key from environment.
	apiKey := os.Getenv("COMICVINE_API_KEY")
	if apiKey == "" {
		return codeError(exitError, comicvine.ErrNoAPIKey)
	}
	a.cfg.APIKey = apiKey

	var err error
	a.client, err = comicvine.NewClient(a.cfg)
	if err != nil {
		return codeError(exitError, err)
	}
	return nil
}

func (a *App) render(records any) error {
	r := NewRenderer(os.Stdout, Format(a.output), a.fields, a.noHeader, a.template)
	return r.Render(records)
}

func (a *App) renderOrEmpty(records any, n int) error {
	if err := a.render(records); err != nil {
		return err
	}
	if n == 0 {
		return codeError(exitNoData, nil)
	}
	return nil
}

func (a *App) progressf(format string, args ...any) {
	if a.quiet {
		return
	}
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

func (a *App) effectiveLimit(def int) int {
	if a.limit > 0 {
		return a.limit
	}
	return def
}

func mapFetchErr(err error) error {
	if err == nil {
		return nil
	}
	if isNotFound(err) {
		return codeError(exitNoData, err)
	}
	if isRateLimited(err) {
		return codeError(exitRateLimited, err)
	}
	// Unwrap ExitError if setup already wrapped it.
	var ee *ExitError
	if errors.As(err, &ee) {
		return err
	}
	return codeError(exitError, err)
}
