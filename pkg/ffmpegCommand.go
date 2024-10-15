package pkg

import (
	"fmt"
	"os/exec"
)

// runCommand runs the provided command and returns an error if it fails.
// It uses the os/exec package to execute the command and checks if
// the command fails. In case of failure, it returns a formatted error
// message containing the command that failed and the corresponding error.
func runCommand(cmd *exec.Cmd) error {
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command: %s, %s", cmd.String(), err)
	}
	return nil
}

// ConvertVideo converts a video file to HLS format (HTTP Live Streaming) using ffmpeg.
// It accepts the video file path (videoPath), the output directory for the converted video (outputPath),
// and an optional quality parameter (quality). If a quality value between 40 and 100 is provided,
// it adjusts the video and audio bitrates accordingly. If no quality is specified, the video retains
// its existing quality. The function generates a playlist (index.m3u8) and segments the video into
// 10-second chunks.
func ConvertVideo(videoPath, outputPath string, quality ...int) error {
	var args []string

	// Add input video file, video codec (libx264), and audio codec (aac) to the arguments
	args = append(args, "-i", videoPath, "-codec:v", "libx264", "-codec:a", "aac")

	// If quality is specified, calculate the video and audio bitrates based on the quality value
	if len(quality) > 0 {
		q := quality[0]
		videoBitrate := fmt.Sprintf("%dk", 500+(q-40)*15) // Adjust video bitrate dynamically
		audioBitrate := fmt.Sprintf("%dk", 64+(q-40)*2)   // Adjust audio bitrate dynamically
		args = append(args, "-b:v", videoBitrate, "-b:a", audioBitrate)
	}

	// Add arguments specific to HLS (HTTP Live Streaming) format
	args = append(args,
		"-hls_time", "10", // Split video into segments of 10 seconds each
		"-hls_playlist_type", "vod", // Define the playlist as Video on Demand (VOD)
		"-hls_segment_filename", fmt.Sprintf("%s/segment%%03d.ts", outputPath), // Define segment file name pattern
		"-start_number", "0", // Start segment numbering from 0
		fmt.Sprintf("%s/index.m3u8", outputPath), // Output the HLS playlist file as index.m3u8
	)

	// Execute the ffmpeg command with the constructed arguments
	return runCommand(exec.Command("ffmpeg", args...))
}

// heights is a map that associates common video resolution heights with their corresponding widths.
// This is used to scale video resolutions during conversion in the ConvertVideoResolutions function.
var heights = map[string]string{"360": "740", "480": "854", "720": "1280", "1080": "1920"}

// ConvertVideoResolutions converts a video file to a specific resolution using ffmpeg.
// It takes the video file path (videoPath), the output directory (outputPath), and the desired resolution.
// The video is scaled to the specified resolution using a video filter and converted to HLS format.
func ConvertVideoResolutions(videoPath, outputPath string, resolution string) error {
	return runCommand(exec.Command("ffmpeg",
		"-i", videoPath, // Input video file path
		"-codec:v", "libx264", // Use the H.264 video codec
		"-codec:a", "aac", // Use AAC for audio codec
		"-vf", fmt.Sprintf("scale=%s:%s", heights[resolution], resolution), // Scale the video to the specified resolution
		"-hls_time", "10", // Split video into 10-second segments
		"-hls_playlist_type", "vod", // Define playlist as Video on Demand (VOD)
		"-hls_segment_filename", fmt.Sprintf("%s/segment%%03d.ts", outputPath), // Define segment file name pattern
		"-start_number", "0", // Start segment numbering from 0
		fmt.Sprintf("%s/index.m3u8", outputPath), // Output the HLS playlist file
	))
}

// ConvertImage converts an image file using ffmpeg by applying compression.
// It accepts the image file path (imagePath), the output file path (outputPath), and the compression level.
// Compression values range from 1 (highest quality) to 31 (lowest quality).
// The function applies the specified compression and generates the output image.
func ConvertImage(imagePath, outputPath, compression string) error {
	return runCommand(exec.Command("ffmpeg",
		"-i", imagePath, // Input image file
		"-q:v", compression, // Set the image compression level
		outputPath, // Output image file path
	))
}

// ConvertAudio converts an audio file to a standard format using ffmpeg.
// It accepts the audio file path (audioPath), the output file path (outputPath), and an optional audio bitrate.
// The audio is converted with a sample rate of 44100 Hz and 2 channels (stereo). If a bitrate is specified,
// it is used in the conversion; otherwise, ffmpeg defaults are applied.
// The function can handle various audio bitrates, such as 128 Kbps for low quality, 192 Kbps for standard quality,
// 256 Kbps for high quality, and 320 Kbps for maximum quality.
func ConvertAudio(audioPath, outputPath string, bitrate ...string) error {
	// Prepare the base arguments for the ffmpeg command
	args := []string{
		"-i", audioPath, // Input audio file path
		"-vn",          // Disable video processing (audio-only conversion)
		"-ar", "44100", // Set the audio sample rate to 44100 Hz
		"-ac", "2", // Set the number of audio channels to 2 (stereo)
	}

	// Append the bitrate option if the user provides one
	if len(bitrate) > 0 {
		args = append(args, "-b:a", bitrate[0]) // Set the specified audio bitrate
	}

	// Add the output file path to the command arguments
	args = append(args, outputPath)

	// Execute the ffmpeg command with the constructed arguments
	return runCommand(exec.Command("ffmpeg", args...))
}
