package pkg

import (
	"fmt"
	"os/exec"
	"strconv"
	"sync"

	"github.com/rs/zerolog/log"
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

var heights = map[int64]string{360: "740", 480: "854", 720: "1280", 1080: "1920"}

func convertVideoResolution(videoPath, outputPath, hlsPath string, resolution int64, wg *sync.WaitGroup, resolutionError *bool) {

	defer wg.Done()

	// Execute the ffmpeg command
	cmd := exec.Command("ffmpeg",
		"-i", videoPath,
		"-codec:v", "libx264",
		"-codec:a", "aac",
		"-vf", fmt.Sprintf("scale=%s:%s", heights[resolution], strconv.FormatInt(resolution, 10)),
		"-hls_time", "10",
		"-hls_playlist_type", "vod",
		"-hls_segment_filename", fmt.Sprintf("%s/segment%%03d.ts", outputPath),
		"-start_number", "0",
		hlsPath,
	)

	if err := cmd.Run(); err != nil {
		log.Error().Msg(fmt.Sprintf("error converting video resolution for "+videoPath+", resolution: %s, error: "+err.Error(), strconv.FormatInt(resolution, 10)))
		*resolutionError = true
	}
}

type FFmpegConfig struct {
	OutputPath string
	HlsPath    string
	Resolution int64
}

func ConvertVideoResolutions(videoPath string, resolutions []FFmpegConfig) error {
	var wg = sync.WaitGroup{}
	var resolutionError = false

	for _, v := range resolutions {
		wg.Add(1)
		go convertVideoResolution(videoPath, v.OutputPath, v.HlsPath, v.Resolution, &wg, &resolutionError)
	}

	wg.Wait()

	if resolutionError {
		return fmt.Errorf("error while converting video resolutions") // Return the first error encountered
	}

	return nil
}
