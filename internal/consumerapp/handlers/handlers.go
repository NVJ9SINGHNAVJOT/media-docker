// Package handlers implements the per-topic conversion work of the consumer
// services.
//
// Every handler follows the same contract, which is what makes v4's stable URLs
// safe:
//
//  1. If the converted output already exists, do nothing. Kafka delivery is
//     at-least-once, so a redelivered message must not re-run ffmpeg.
//  2. Convert into the asset's scratch directory, never into its served paths.
//  3. On success, promote the output with a rename.
//  4. On failure, remove the scratch directory.
//
// The raw upload is never removed, on either path. Conversion adds a
// representation rather than replacing one: the asset's URL upgrades from the
// raw upload to the converted output, and the raw upload stays on disk behind
// the asset's VariantOriginal URL, /media/<types>/<id>/original.
//
// Keeping it on failure is what makes a permanently failed conversion degrade
// quality rather than break an asset, since the URL a caller already holds keeps
// resolving. Keeping it on success is what makes the source bytes retrievable
// for the life of the asset, and it means a redelivered or retried message
// always still has its input.
//
// The only thing that removes an asset is an explicit delete, which takes the
// whole directory -- metadata, raw upload and converted output together.
package handlers

import (
	"fmt"
	"os"
	"strconv"

	"github.com/nvj9singhnavjot/media-docker/pkg"
	"github.com/nvj9singhnavjot/media-docker/pkg/asset"
	"github.com/rs/zerolog/log"
)

// itoa formats an integer for use as an ffmpeg argument.
func itoa(v int) string {
	return strconv.Itoa(v)
}

// prepareProcessing clears any leftover scratch directory and creates a fresh
// one. Leftovers are possible after a crash mid-conversion.
func prepareProcessing(mediaType, id string) (string, error) {
	dir := asset.ProcessingDir(mediaType, id)

	if err := os.RemoveAll(dir); err != nil {
		return "", fmt.Errorf("error clearing processing directory: %w", err)
	}

	if err := pkg.CreateDir(dir); err != nil {
		return "", fmt.Errorf("error creating processing directory: %w", err)
	}

	return dir, nil
}

// removeProcessing discards the scratch directory once it is no longer needed,
// whether the conversion succeeded or failed.
//
// Removal is synchronous rather than queued. The failed consumer retries a
// conversion immediately after a failure, and a queued delete could land after
// the retry had already recreated the directory, deleting output from a run in
// progress.
func removeProcessing(dir string) {
	if err := os.RemoveAll(dir); err != nil {
		log.Error().Err(err).Str("dir", dir).Msg("Error removing processing directory")
	}
}

// exists reports whether a path is present, treating errors as absence.
func exists(path string) bool {
	ok, err := pkg.DirOrFileExist(path)
	return err == nil && ok
}
