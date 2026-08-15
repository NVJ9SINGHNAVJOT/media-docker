package api

import (
	"os"
	"path/filepath"
	"time"

	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/pkg/asset"
	"github.com/rs/zerolog/log"
)

// Janitor sweep settings.
const (
	// janitorInterval is how often abandoned uploads are swept for.
	janitorInterval = time.Hour

	// janitorTTL is how long an upload may sit unclaimed before it is removed.
	// It needs to comfortably exceed the time a slow client takes to finish a
	// large chunked upload and call a dispatch endpoint.
	janitorTTL = 6 * time.Hour
)

// StartJanitor runs a periodic sweep for uploads that were never claimed by a
// dispatch endpoint.
//
// Writing the raw upload straight into media storage is what lets a URL work
// immediately, but it also means a client that uploads and then walks away
// leaves a file in the publicly served tree. The dispatched flag distinguishes
// the two cases, and this sweep removes what was abandoned. It also clears
// orphaned chunk staging directories, which previously accumulated with nothing
// to clean them up.
//
// NOTE: It is important to call this function within a goroutine.
func StartJanitor() {
	// Sweep once at startup to clear anything left behind by a previous run.
	sweep()

	ticker := time.NewTicker(janitorInterval)
	defer ticker.Stop()

	for range ticker.C {
		sweep()
	}
}

// sweep removes abandoned assets and orphaned staging directories.
func sweep() {
	cutoff := time.Now().Add(-janitorTTL)

	for _, mediaType := range []string{
		asset.TypeVideo, asset.TypeImage, asset.TypeAudio, asset.TypeDocument, asset.TypeOther,
	} {
		sweepAssets(mediaType, cutoff)
		sweepStaging(mediaType, cutoff)
	}
}

// sweepAssets removes asset directories that were never dispatched and have
// aged past the cutoff.
func sweepAssets(mediaType string, cutoff time.Time) {
	root := filepath.Join(helper.Constants.MediaStorage, asset.TypeDir(mediaType))

	entries, err := os.ReadDir(root)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Error().Err(err).Str("dir", root).Msg("Janitor could not read media directory")
		}
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		meta, err := asset.ReadMeta(mediaType, entry.Name())
		if err != nil {
			// An asset directory with no readable metadata cannot be resolved
			// and cannot be reasoned about; remove it once it is old enough.
			if info, statErr := entry.Info(); statErr == nil && info.ModTime().Before(cutoff) {
				removeAbandoned(filepath.Join(root, entry.Name()), "unreadable metadata")
			}
			continue
		}

		if meta.Dispatched || meta.CreatedAt.After(cutoff) {
			continue
		}

		removeAbandoned(filepath.Join(root, entry.Name()), "upload never dispatched")
	}
}

// sweepStaging removes chunk staging directories left behind by uploads that
// were started but never completed.
func sweepStaging(mediaType string, cutoff time.Time) {
	root := filepath.Join(helper.Constants.UploadStorage, asset.TypeDir(mediaType))

	entries, err := os.ReadDir(root)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Error().Err(err).Str("dir", root).Msg("Janitor could not read staging directory")
		}
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil || !info.ModTime().Before(cutoff) {
			continue
		}

		removeAbandoned(filepath.Join(root, entry.Name()), "chunk upload never completed")
	}
}

// removeAbandoned deletes a directory and logs the outcome.
//
// Deletion is synchronous rather than queued through the delete channels: the
// sweep is already off the request path, and queuing would silently drop work
// whenever the buffered channel is full.
func removeAbandoned(path, reason string) {
	if err := os.RemoveAll(path); err != nil {
		log.Error().Err(err).Str("path", path).Str("reason", reason).Msg("Janitor failed to remove abandoned upload")
		return
	}

	log.Info().Str("path", path).Str("reason", reason).Msg("Janitor removed abandoned upload")
}
