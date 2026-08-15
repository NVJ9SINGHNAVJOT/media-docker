package pkg

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ffmpegInstallHint is appended to every ffmpeg availability error. Without it a
// missing binary surfaces only as every job failing to the dead-letter queue,
// which is what it looked like the first time this happened.
const ffmpegInstallHint = "install ffmpeg and make sure it is on PATH " +
	"(macOS: `brew install ffmpeg`, Debian/Ubuntu: `apt-get install ffmpeg`; " +
	"in Docker the consumer images build from internal/Dockerfile.ffmpeg)"

// CheckFFmpeg verifies that ffmpeg is installed and actually executable,
// returning its version line for logging.
func CheckFFmpeg() (string, error) {
	path, err := exec.LookPath("ffmpeg")
	if err != nil {
		return "", fmt.Errorf("ffmpeg not found: %v, %s", err, ffmpegInstallHint)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	output, err := exec.CommandContext(ctx, path, "-version").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ffmpeg at %s is not executable: %v, %s", path, err, ffmpegInstallHint)
	}

	// The first line looks like "ffmpeg version 7.1 Copyright (c) ...".
	version := string(output)
	if idx := strings.IndexByte(version, '\n'); idx >= 0 {
		version = version[:idx]
	}

	return strings.TrimSpace(version), nil
}
