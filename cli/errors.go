package cli

import (
	"errors"

	"github.com/tamnd/comicvine-cli/comicvine"
)

func isNotFound(err error) bool {
	return errors.Is(err, comicvine.ErrNotFound)
}

func isRateLimited(err error) bool {
	return errors.Is(err, comicvine.ErrRateLimited)
}
