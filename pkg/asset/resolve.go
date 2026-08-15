package asset

import (
	"errors"
	"path/filepath"

	"github.com/nvj9singhnavjot/media-docker/pkg"
)

// ErrNotFound indicates that no asset exists for the requested type and id,
// or that the asset directory exists but holds nothing servable.
var ErrNotFound = errors.New("asset not found")

// Resolution is the outcome of resolving an asset to the best representation
// currently available on disk.
type Resolution struct {
	// RelPath is the file to serve, relative to the media storage root,
	// e.g. "videos/3f2b.../hls/index.m3u8". It is always slash-separated so it
	// can be appended to a URL directly.
	RelPath string

	// Converted reports whether RelPath points at processed output rather than
	// the raw upload. Callers use this to decide cache headers: an unconverted
	// asset may change representation at any moment.
	Converted bool

	// Meta is the asset's metadata, needed for the download file name of
	// document and other assets.
	Meta Meta
}

// Resolve picks the best representation of an asset that currently exists.
//
// This is the mechanism behind v4's non-blocking URLs. A single URL resolves to
// the raw upload while conversion is pending or has failed, and to the converted
// output once a consumer has promoted it into place. Because promotion is a
// rename, there is no intermediate state where a half-written playlist is
// selected.
//
// variant selects a rung of the resolution ladder for video assets ("360",
// "480", "720", "1080"); pass an empty string for the default representation,
// which for a video-resolutions asset is the master playlist.
//
// Falling back to the raw upload is not an error condition. It is the normal
// state between upload and conversion, and the permanent state for an asset
// whose conversion failed, which is why a failed conversion no longer breaks a
// URL that was already handed to a caller.
func Resolve(mediaType, id, variant string) (Resolution, error) {
	if !IsStorageType(mediaType) {
		return Resolution{}, ErrNotFound
	}

	meta, err := ReadMeta(mediaType, id)
	if err != nil {
		return Resolution{}, err
	}

	relDir := RelDir(mediaType, id)

	switch mediaType {
	case TypeVideo:
		// A specific rung of the ladder, when one was asked for and exists.
		if IsResolution(variant) {
			if ok, err := pkg.DirOrFileExist(ResolutionPlaylistPath(id, variant)); err == nil && ok {
				return Resolution{
					RelPath:   join(relDir, HLSDirName, variant, PlaylistName),
					Converted: true,
					Meta:      meta,
				}, nil
			}
		}

		// The top-level playlist: a single-quality playlist for a video job, or
		// the master playlist listing the ladder for a video-resolutions job.
		if ok, err := pkg.DirOrFileExist(PlaylistPath(id)); err == nil && ok {
			return Resolution{
				RelPath:   join(relDir, HLSDirName, PlaylistName),
				Converted: true,
				Meta:      meta,
			}, nil
		}

	case TypeImage, TypeAudio:
		converted, err := ConvertedPath(mediaType, id)
		if err != nil {
			return Resolution{}, err
		}
		if ok, err := pkg.DirOrFileExist(converted); err == nil && ok {
			return Resolution{
				RelPath:   join(relDir, filepath.Base(converted)),
				Converted: true,
				Meta:      meta,
			}, nil
		}

	case TypeDocument, TypeOther:
		// Never converted; always served as uploaded.
	}

	// Fall back to the raw upload.
	ok, err := pkg.DirOrFileExist(OriginalPath(mediaType, id, meta.Ext))
	if err != nil || !ok {
		return Resolution{}, ErrNotFound
	}

	return Resolution{
		RelPath:   join(relDir, OriginalName(meta.Ext)),
		Converted: false,
		Meta:      meta,
	}, nil
}

// join builds a slash-separated relative path suitable for use in a URL.
func join(parts ...string) string {
	return filepath.ToSlash(filepath.Join(parts...))
}
