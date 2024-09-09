package pkg

import (
	"fmt"
	"os/exec"
)

// ConvertVideo return pointer to the Cmd struct to execute the
// video conversion command with video file path, output path, and if quality given.
func ConvertVideo(videoPath, outputPath string, quality ...int) *exec.Cmd {
	var args []string

	args = append(args, "-i", videoPath, "-codec:v", "libx264", "-codec:a", "aac")

	if len(quality) > 0 {
		q := quality[0]
		videoBitrate := fmt.Sprintf("%dk", 500+(q-40)*15)
		audioBitrate := fmt.Sprintf("%dk", 64+(q-40)*2)
		args = append(args, "-b:v", videoBitrate, "-b:a", audioBitrate)
	}

	args = append(args,
		"-hls_time", "10",
		"-hls_playlist_type", "vod",
		"-hls_segment_filename", fmt.Sprintf("%s/segment%%03d.ts", outputPath),
		"-start_number", "0",
		fmt.Sprintf("%s/index.m3u8", outputPath),
	)

	return exec.Command("ffmpeg", args...)
}

var heights = map[string]string{"360": "740", "480": "854", "720": "1280", "1080": "1920"}

func ConvertVideoResolution(videoPath, outputPath string, resolution string) *exec.Cmd {
	return exec.Command("ffmpeg",
		"-i", videoPath,
		"-codec:v", "libx264",
		"-codec:a", "aac",
		"-vf", fmt.Sprintf("scale=%s:%s", heights[resolution], resolution),
		"-hls_time", "10",
		"-hls_playlist_type", "vod",
		"-hls_segment_filename", fmt.Sprintf("%s/segment%%03d.ts", outputPath),
		"-start_number", "0",
		fmt.Sprintf("%s/index.m3u8", outputPath),
	)
}

// can adjust the value (1 to 31) for compression
func ConvertImage(imagePath, outputPath, compression string) *exec.Cmd {
	return exec.Command("ffmpeg",
		"-i", imagePath,
		"-q:v", compression,
		outputPath,
	)
}

// 320 Kbps for max audio quality
func ConvertAudio(audioPath, outputPath string) *exec.Cmd {
	return exec.Command("ffmpeg",
		"-i", audioPath,
		"-vn",
		"-ar", "44100",
		"-ac", "2",
		"-b:a", "192k",
		outputPath,
	)
}
