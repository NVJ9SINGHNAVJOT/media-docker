package pkg

import (
	"fmt"
	"os/exec"
)

func ConvertVideo(videoPath, outputPath, hlsPath string) error {

	// Execute the ffmpeg command
	cmd := exec.Command("ffmpeg",
		"-i", videoPath,
		"-codec:v", "libx264",
		"-codec:a", "aac",
		"-hls_time", "10",
		"-hls_playlist_type", "vod",
		"-hls_segment_filename", fmt.Sprintf("%s/segment%%03d.ts", outputPath),
		"-start_number", "0",
		hlsPath,
	)

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}
