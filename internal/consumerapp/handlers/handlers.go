// Package handlers implements the per-topic conversion work of the consumer
// services.
//
// Every handler follows the same contract, which is what makes v4's stable URLs
// safe:
//
//  1. If the converted output already exists, do nothing. Kafka delivery is
//     at-least-once, so a redelivered message must not re-run ffmpeg -- and by
//     that point the raw input has usually been removed anyway.
//  2. Convert into the asset's scratch directory, never into its served paths.
//  3. On success, promote the output with a rename and only then remove the raw
//     upload.
//  4. On failure, remove the scratch directory and leave the raw upload alone.
//
// Step 4 is the important one. The raw upload is what the asset's URL is serving
// while conversion is pending, so deleting it on failure would break a URL a
// caller is already using. Leaving it means a permanently failed conversion
// degrades quality rather than breaking the asset.
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
// After a failure the raw upload is deliberately left in place: it is the
// representation the asset's URL currently resolves to.
//
// Removal is synchronous rather than queued through the delete channels. The
// failed consumer retries a conversion immediately after a failure, and a queued
// delete could land after the retry had already recreated the directory,
// deleting output from a run in progress.
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
