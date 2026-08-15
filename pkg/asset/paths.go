// Package asset owns the on-disk layout of every stored media asset.
//
// Every asset is a directory rather than a bare file:
//
//	media_docker_files/videos/<id>/
//	  meta.json          metadata written at upload time
//	  original.mp4       the raw upload, served immediately
//	  .processing/       conversion scratch space, never served
//	  hls/index.m3u8     the converted output, appears atomically
//
// This uniformity is what allows a single stable URL to serve the raw upload
// first and the converted output later (see Resolve), and it makes deletion a
// single RemoveAll regardless of media type.
//
// The server, the client, and every consumer import this package so that path
// knowledge lives in exactly one place.
package asset

import (
	"fmt"
	"path/filepath"

	"github.com/nvj9singhnavjot/media-docker/helper"
)

// Storage categories. These are the values accepted in the "type" field of an
// upload and they determine the directory an asset lives in.
//
// Note that "videoResolutions" is deliberately absent: it is a processing
// variant of TypeVideo, not a storage category, and its output lives in the
// same videos/<id>/ directory as a single-quality video.
const (
	TypeVideo    = "video"
	TypeImage    = "image"
	TypeAudio    = "audio"
	TypeDocument = "document"
	TypeOther    = "other"
)

// File and directory names used inside an asset directory.
const (
	MetaName          = "meta.json"
	OriginalStem      = "original"
	ProcessingDirName = ".processing"
	HLSDirName        = "hls"
	PlaylistName      = "index.m3u8"

	// Converted image and audio output always use a fixed extension, because
	// ConvertImage always produces JPEG and ConvertAudio always produces MP3.
	ConvertedImageName = "converted.jpeg"
	ConvertedAudioName = "converted.mp3"
)

// Rung describes one step of the video-resolutions ladder.
//
// Bandwidth is the value advertised in the master playlist. It only needs to be
// a reasonable estimate: players use it to order and select variants, not to
// allocate anything.
type Rung struct {
	Name      string // Directory name and public variant name, e.g. "720"
	Width     int
	Height    int
	Bandwidth int
}

// Ladder is the single source of truth for the video-resolutions output,
// ordered from lowest to highest quality (the order a master playlist lists
// them in). Both the ffmpeg scale filter and the master playlist derive from it.
var Ladder = []Rung{
	{Name: "360", Width: 640, Height: 360, Bandwidth: 800_000},
	{Name: "480", Width: 854, Height: 480, Bandwidth: 1_400_000},
	{Name: "720", Width: 1280, Height: 720, Bandwidth: 2_800_000},
	{Name: "1080", Width: 1920, Height: 1080, Bandwidth: 5_000_000},
}

// Resolutions lists the ladder's variant names.
var Resolutions = func() []string {
	names := make([]string, len(Ladder))
	for i, rung := range Ladder {
		names[i] = rung.Name
	}
	return names
}()

// Scale returns the ffmpeg scale filter argument for a rung, e.g. "640:360".
func (r Rung) Scale() string {
	return fmt.Sprintf("%d:%d", r.Width, r.Height)
}

// IsResolution reports whether v names one of the resolutions in the ladder.
func IsResolution(v string) bool {
	for _, r := range Ladder {
		if r.Name == v {
			return true
		}
	}
	return false
}

// IsStorageType reports whether mediaType is a valid storage category.
func IsStorageType(mediaType string) bool {
	switch mediaType {
	case TypeVideo, TypeImage, TypeAudio, TypeDocument, TypeOther:
		return true
	}
	return false
}

// TypeDir returns the directory name holding all assets of a category,
// relative to the media storage root: "video" -> "videos".
func TypeDir(mediaType string) string {
	return mediaType + "s"
}

// RelDir returns an asset's directory relative to the media storage root,
// e.g. "videos/3f2b...".
func RelDir(mediaType, id string) string {
	return filepath.ToSlash(filepath.Join(TypeDir(mediaType), id))
}

// Dir returns an asset's directory as a path usable on disk,
// e.g. "media_docker_files/videos/3f2b...".
func Dir(mediaType, id string) string {
	return filepath.Join(helper.Constants.MediaStorage, TypeDir(mediaType), id)
}

// MetaPath returns the path of an asset's meta.json.
func MetaPath(mediaType, id string) string {
	return filepath.Join(Dir(mediaType, id), MetaName)
}

// OriginalName returns the file name of the raw upload for a given extension.
func OriginalName(ext string) string {
	return OriginalStem + "." + ext
}

// OriginalPath returns the path of the raw upload.
func OriginalPath(mediaType, id, ext string) string {
	return filepath.Join(Dir(mediaType, id), OriginalName(ext))
}

// ProcessingDir returns the scratch directory a consumer writes conversion
// output into. Nothing under this directory is ever served: output only becomes
// visible when it is promoted into place (see PromoteDir and PromoteFile).
func ProcessingDir(mediaType, id string) string {
	return filepath.Join(Dir(mediaType, id), ProcessingDirName)
}

// HLSDir returns the directory holding the converted HLS output of a video.
func HLSDir(id string) string {
	return filepath.Join(Dir(TypeVideo, id), HLSDirName)
}

// PlaylistPath returns the path of the top-level HLS playlist of a video.
// For a video-resolutions asset this is the master playlist listing the ladder.
func PlaylistPath(id string) string {
	return filepath.Join(HLSDir(id), PlaylistName)
}

// ResolutionPlaylistPath returns the path of the playlist for one rung of the
// resolution ladder, e.g. hls/720/index.m3u8.
func ResolutionPlaylistPath(id, resolution string) string {
	return filepath.Join(HLSDir(id), resolution, PlaylistName)
}

// ResolvePrefix is the URL prefix served by the client's resolver.
//
// It is deliberately separate from the static media storage mount: a resolver
// pattern such as videos/{id}/{variant} would otherwise also match real file
// paths like videos/<id>/hls, making routing ambiguous.
const ResolvePrefix = "/media"

// URLPath returns the stable, public path of an asset,
// e.g. "/media/videos/3f2b...".
//
// This is the URL handed to callers. It never changes for the life of the
// asset, while what it resolves to upgrades from the raw upload to the
// converted output once a consumer finishes.
func URLPath(mediaType, id string) string {
	return ResolvePrefix + "/" + TypeDir(mediaType) + "/" + id
}

// VariantURLPath returns the public path of one rung of a video's resolution
// ladder, e.g. "/media/videos/3f2b.../720".
func VariantURLPath(mediaType, id, variant string) string {
	return URLPath(mediaType, id) + "/" + variant
}

// StagingDir returns the private directory chunks are written to while an
// upload is in progress. It lives on the upload storage volume and is never
// served; only the merged result is written into media storage.
func StagingDir(mediaType, id string) string {
	return filepath.Join(helper.Constants.UploadStorage, TypeDir(mediaType), id)
}

// ConvertedPath returns the path of the converted output for image and audio
// assets. It returns an error for any other category, which has no single
// converted file.
func ConvertedPath(mediaType, id string) (string, error) {
	switch mediaType {
	case TypeImage:
		return filepath.Join(Dir(mediaType, id), ConvertedImageName), nil
	case TypeAudio:
		return filepath.Join(Dir(mediaType, id), ConvertedAudioName), nil
	default:
		return "", fmt.Errorf("no converted file for media type %q", mediaType)
	}
}
