package asset

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// PromoteDir makes a directory of conversion output visible in one atomic step.
//
// Consumers write into ProcessingDir and promote on success. This matters
// because ffmpeg builds an HLS playlist incrementally: index.m3u8 exists, and is
// incomplete, for the whole duration of a transcode. Resolving on the existence
// of that file while ffmpeg was still writing would hand clients a truncated
// playlist. Promotion by rename means the converted output either is not there
// at all or is there complete.
//
// If final already exists (a re-processed asset), it is moved aside first and
// removed afterwards, keeping the window in which neither is in place as short
// as a single rename.
func PromoteDir(tmp, final string) error {
	if _, err := os.Stat(tmp); err != nil {
		return fmt.Errorf("cannot promote %s: %w", tmp, err)
	}

	var superseded string

	if _, err := os.Stat(final); err == nil {
		superseded = final + ".superseded-" + strconv.FormatInt(time.Now().UnixNano(), 10)
		if err = os.Rename(final, superseded); err != nil {
			return fmt.Errorf("error moving aside existing output %s: %w", final, err)
		}
	}

	if err := os.Rename(tmp, final); err != nil {
		// Put the previous output back so the asset keeps serving something.
		if superseded != "" {
			if restoreErr := os.Rename(superseded, final); restoreErr != nil {
				return fmt.Errorf("error promoting %s (%v) and restoring previous output failed: %w", tmp, err, restoreErr)
			}
		}
		return fmt.Errorf("error promoting %s to %s: %w", tmp, final, err)
	}

	if superseded != "" {
		if err := os.RemoveAll(superseded); err != nil {
			// The asset is correct; only the old copy lingers.
			return fmt.Errorf("promoted %s but failed to remove superseded output: %w", final, err)
		}
	}

	return nil
}

// PromoteFile makes a single converted file visible atomically.
// On POSIX a rename over an existing file is atomic, so no move-aside is needed.
func PromoteFile(tmp, final string) error {
	if err := os.Rename(tmp, final); err != nil {
		return fmt.Errorf("error promoting %s to %s: %w", tmp, final, err)
	}
	return nil
}

// WriteMasterPlaylist writes an HLS master playlist listing every rung of the
// ladder that was actually produced in dir.
//
// Without this, a video-resolutions asset would have no top-level playlist and
// its base URL would have nothing to resolve to. It also delivers the adaptive
// quality switching the project has always advertised: players read the master
// playlist and pick a variant from the available bandwidth.
//
// dir is the directory holding the per-resolution subdirectories, normally the
// processing directory, so the playlist is promoted along with the output.
func WriteMasterPlaylist(dir string) error {
	var b strings.Builder

	b.WriteString("#EXTM3U\n")
	b.WriteString("#EXT-X-VERSION:3\n")

	written := 0
	for _, rung := range Ladder {
		// Only advertise variants that exist, so a partially produced ladder
		// still yields a valid playlist.
		if _, err := os.Stat(filepath.Join(dir, rung.Name, PlaylistName)); err != nil {
			continue
		}

		fmt.Fprintf(&b, "#EXT-X-STREAM-INF:BANDWIDTH=%d,RESOLUTION=%dx%d\n", rung.Bandwidth, rung.Width, rung.Height)
		fmt.Fprintf(&b, "%s/%s\n", rung.Name, PlaylistName)
		written++
	}

	if written == 0 {
		return fmt.Errorf("no resolution playlists found in %s", dir)
	}

	if err := os.WriteFile(filepath.Join(dir, PlaylistName), []byte(b.String()), 0644); err != nil {
		return fmt.Errorf("error writing master playlist in %s: %w", dir, err)
	}

	return nil
}
