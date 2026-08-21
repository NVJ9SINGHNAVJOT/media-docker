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

	// Immutable reports whether the selected representation is terminal: nothing
	// a consumer does later will replace it. Callers use this to decide cache
	// headers.
	//
	// It is true for converted output, for an explicitly requested original, and
	// for categories that are never converted at all. It is false only where the
	// raw upload was selected as a fallback and a conversion could still promote
	// output over it.
	Immutable bool

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
// variant selects a specific representation instead of the best one. It accepts
// a rung of the resolution ladder for video assets ("360", "480", "720",
// "1080"), or VariantOriginal for any category, which pins the result to the raw
// upload. Pass an empty string for the default representation, which for a
// video-resolutions asset is the master playlist.
//
// Because the raw upload is never removed, VariantOriginal resolves for the
// whole life of an asset, not only until its conversion finishes.
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

	// An explicit request for the source bytes skips the converted output
	// entirely, and is terminal: this is the one representation that can never
	// be superseded.
	if variant == VariantOriginal {
		return originalResolution(mediaType, id, relDir, meta, true)
	}

	switch mediaType {
	case TypeVideo:
		// A specific rung of the ladder, when one was asked for and exists.
		if IsResolution(variant) {
			if ok, err := pkg.DirOrFileExist(ResolutionPlaylistPath(id, variant)); err == nil && ok {
				return Resolution{
					RelPath:   join(relDir, HLSDirName, variant, PlaylistName),
					Immutable: true,
					Meta:      meta,
				}, nil
			}
		}

		// The top-level playlist: a single-quality playlist for a video job, or
		// the master playlist listing the ladder for a video-resolutions job.
		if ok, err := pkg.DirOrFileExist(PlaylistPath(id)); err == nil && ok {
			return Resolution{
				RelPath:   join(relDir, HLSDirName, PlaylistName),
				Immutable: true,
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
				Immutable: true,
				Meta:      meta,
			}, nil
		}

	case TypeDocument, TypeOther:
		// Never converted; always served as uploaded.
	}

	// Fall back to the raw upload. For a category that is never converted this
	// is not a fallback at all but the asset's only, and final, representation.
	neverConverted := mediaType == TypeDocument || mediaType == TypeOther

	return originalResolution(mediaType, id, relDir, meta, neverConverted)
}

// originalResolution selects the raw upload, if it is still on disk.
func originalResolution(mediaType, id, relDir string, meta Meta, immutable bool) (Resolution, error) {
	ok, err := pkg.DirOrFileExist(OriginalPath(mediaType, id, meta.Ext))
	if err != nil || !ok {
		return Resolution{}, ErrNotFound
	}

	return Resolution{
		RelPath:   join(relDir, OriginalName(meta.Ext)),
		Immutable: immutable,
		Meta:      meta,
	}, nil
}

// join builds a slash-separated relative path suitable for use in a URL.
func join(parts ...string) string {
	return filepath.ToSlash(filepath.Join(parts...))
}
