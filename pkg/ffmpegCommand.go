package pkg

import (
	"fmt"
	// "io"
	// "os"
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

	// NOTE: Used only development.
	// Set the command's stdout and stderr to os.Stdout and os.Stderr
	// so they will be printed in real-time.
	// stdout, err := cmd.StdoutPipe()
	// if err != nil {
	// 	return fmt.Errorf("failed to get stdout: %w", err)
	// }

	// stderr, err := cmd.StderrPipe()
	// if err != nil {
	// 	return fmt.Errorf("failed to get stderr: %w", err)
	// }

	// // Start the command before consuming its output
	// if err := cmd.Start(); err != nil {
	// 	return fmt.Errorf("failed to start command: %w", err)
	// }

	// // Stream the command's stdout
	// go io.Copy(os.Stdout, stdout)

	// // Stream the command's stderr
	// go io.Copy(os.Stderr, stderr)

	// // Wait for the command to finish and return an error if it fails
	// if err := cmd.Wait(); err != nil {
	// 	return fmt.Errorf("command: %s, %s", cmd.String(), err)
	// }
	// return nil
}

// ConvertVideo converts a video file to HLS format (HTTP Live Streaming) using ffmpeg.
// It accepts the following parameters:
//   - videoPath: the path to the input video file to be converted.
//   - outputPath: the directory where the converted video segments and playlist will be saved.
//   - quality: an optional parameter that adjusts the video and audio bitrates.
//     If a quality value between 40 and 100 is provided, it calculates the corresponding
//     bitrates for video and audio. If no quality is specified, the video retains its existing quality.
//
// The function generates a playlist (index.m3u8) and segments the video into 10-second chunks.
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
// This map is used to scale video resolutions during conversion in the ConvertVideoResolutions function.
var heights = map[string]string{"360": "740", "480": "854", "720": "1280", "1080": "1920"}

// ConvertVideoResolutions converts a video file to a specific resolution using ffmpeg.
// It accepts the following parameters:
//   - videoPath: the path to the input video file to be converted.
//   - outputPath: the directory where the converted video segments and playlist will be saved.
//   - resolution: the desired resolution to which the video will be scaled.
//
// The video is scaled to the specified resolution using a video filter and converted to HLS format.
func ConvertVideoResolutions(videoPath, outputPath string, resolution string) error {
	return runCommand(exec.Command("ffmpeg",
		"-i", videoPath, // Input video file path
		"-codec:v", "libx264", // Use the H.264 video codec for video conversion
		"-codec:a", "aac", // Use AAC for audio codec
		"-vf", fmt.Sprintf("scale=%s:%s", heights[resolution], resolution), // Scale the video to the specified resolution
		"-hls_time", "10", // Split video into 10-second segments
		"-hls_playlist_type", "vod", // Define the playlist as Video on Demand (VOD)
		"-hls_segment_filename", fmt.Sprintf("%s/segment%%03d.ts", outputPath), // Define segment file name pattern
		"-start_number", "0", // Start segment numbering from 0
		fmt.Sprintf("%s/index.m3u8", outputPath), // Output the HLS playlist file
	))
}

// ConvertImage converts an image file using ffmpeg by applying compression.
// It accepts the following parameters:
//   - imagePath: the path to the input image file.
//   - outputPath: the path where the compressed image will be saved.
//   - compression: a string representing the compression level, ranging from 1 (highest quality) to 31 (lowest quality).
//
// The function applies the specified compression level and generates the output image.
func ConvertImage(imagePath, outputPath, compression string) error {
	return runCommand(exec.Command("ffmpeg",
		"-i", imagePath, // Input image file
		"-q:v", compression, // Set the image compression level
		outputPath, // Output image file path
	))
}

// ConvertAudio converts an audio file to a standard format using ffmpeg.
// It accepts the following parameters:
//   - audioPath: the path to the input audio file to be converted.
//   - outputPath: the path where the converted audio file will be saved.
//   - bitrate: an optional parameter to specify the audio bitrate for conversion.
//     If a bitrate is provided, it will be used; otherwise, ffmpeg defaults will apply.
//
// The audio is converted with a sample rate of 44100 Hz and 2 channels (stereo).
// It can handle various audio bitrates such as 128 Kbps for low quality, 192 Kbps for standard quality,
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
