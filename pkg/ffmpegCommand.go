package pkg

import (
	"fmt"
	"os/exec"
)

func ConvertVideo(videoPath, outputPath string) *exec.Cmd {
	return exec.Command("ffmpeg",
		"-i", videoPath,
		"-codec:v", "libx264",
		"-codec:a", "aac",
		"-hls_time", "10",
		"-hls_playlist_type", "vod",
		"-hls_segment_filename", fmt.Sprintf("%s/segment%%03d.ts", outputPath),
		"-start_number", "0",
		fmt.Sprintf("%s/index.m3u8", outputPath),
	)
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
