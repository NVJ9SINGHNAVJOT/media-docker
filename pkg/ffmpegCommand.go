package pkg

import (
	"fmt"
	"os/exec"
)

// ConvertVideo returns a pointer to the Cmd struct to execute the
// video conversion command with the given video file path, output path,
// and an optional quality parameter.
// Quality can be in between 40 and max 100 (preferred),
// if quality is not provided than existing video quality will be used.
func ConvertVideo(videoPath, outputPath string, quality ...int) *exec.Cmd {
	var args []string

	// Add input file and codecs to the arguments
	args = append(args, "-i", videoPath, "-codec:v", "libx264", "-codec:a", "aac")

	// If quality is provided, calculate and add video and audio bitrates
	if len(quality) > 0 {
		q := quality[0]
		videoBitrate := fmt.Sprintf("%dk", 500+(q-40)*15)
		audioBitrate := fmt.Sprintf("%dk", 64+(q-40)*2)
		args = append(args, "-b:v", videoBitrate, "-b:a", audioBitrate)
	}

	// Add HLS (HTTP Live Streaming) specific arguments
	args = append(args,
		"-hls_time", "10", // Set segment duration to 10 seconds
		"-hls_playlist_type", "vod", // Set playlist type to Video on Demand
		"-hls_segment_filename", fmt.Sprintf("%s/segment%%03d.ts", outputPath), // Set segment filename pattern
		"-start_number", "0", // Start segment numbering from 0
		fmt.Sprintf("%s/index.m3u8", outputPath), // Set output playlist file
	)

	// Return the command to execute
	return exec.Command("ffmpeg", args...)
}

// heights maps common video resolutions to their corresponding widths.
var heights = map[string]string{"360": "740", "480": "854", "720": "1280", "1080": "1920"}

// ConvertVideoResolutions returns a pointer to the Cmd struct to execute the
// video conversion command with the given video file path, output path, and resolution.
func ConvertVideoResolutions(videoPath, outputPath string, resolution string) *exec.Cmd {
	return exec.Command("ffmpeg",
		"-i", videoPath, // Input file
		"-codec:v", "libx264", // Video codec
		"-codec:a", "aac", // Audio codec
		"-vf", fmt.Sprintf("scale=%s:%s", heights[resolution], resolution), // Video filter to scale resolution
		"-hls_time", "10", // Set segment duration to 10 seconds
		"-hls_playlist_type", "vod", // Set playlist type to Video on Demand
		"-hls_segment_filename", fmt.Sprintf("%s/segment%%03d.ts", outputPath), // Set segment filename pattern
		"-start_number", "0", // Start segment numbering from 0
		fmt.Sprintf("%s/index.m3u8", outputPath), // Set output playlist file
	)
}

// ConvertImage returns a pointer to the Cmd struct to execute the
// image conversion command with the given image file path, output path,
// and compression level. The compression value can be adjusted from 1 to 31,
// where 1 is the highest quality and 31 is the lowest quality.
func ConvertImage(imagePath, outputPath, compression string) *exec.Cmd {
	return exec.Command("ffmpeg",
		"-i", imagePath, // Input image file
		"-q:v", compression, // Set the compression level for the output image
		outputPath, // Output image file
	)
}

// ConvertAudio returns a pointer to the Cmd struct to execute the
// audio conversion command with the given audio file path, output path, and optional audio bitrate.
// The audio is converted to a standard format with a sample rate of 44100 Hz,
// and 2 audio channels. If bitrate is provided, it's included in the command.
// Otherwise, no bitrate is specified, and ffmpeg uses its default bitrate.
//
// Common audio bitrates (optional):
// - 128 Kbps: Low quality (suitable for spoken audio, podcasts)
// - 192 Kbps: Standard quality (suitable for music and general audio files)
// - 256 Kbps: High quality (better for music with more detail)
// - 320 Kbps: Maximum quality (best for high-fidelity audio)
func ConvertAudio(audioPath, outputPath string, bitrate ...string) *exec.Cmd {
	// Prepare the base command
	args := []string{
		"-i", audioPath, // Input audio file
		"-vn",          // Disable video recording
		"-ar", "44100", // Set audio sample rate to 44100 Hz
		"-ac", "2", // Set number of audio channels to 2 (stereo)
	}

	// Append the bitrate option if provided
	if len(bitrate) > 0 {
		args = append(args, "-b:a", bitrate[0]) // Set audio bitrate
	}

	// Append the output file path
	args = append(args, outputPath)

	// Return the ffmpeg command
	return exec.Command("ffmpeg", args...)
}
