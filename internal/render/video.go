package render

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

type CommandRunner func(ctx context.Context, name string, args ...string) error

type VideoRenderer struct {
	FFmpegPath string
	RunCommand CommandRunner
}

func (r VideoRenderer) Render(ctx context.Context, content VideoContent, outputDir string) ([]OutputFile, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, err
	}

	inputPath := filepath.Join(outputDir, "remotion-input.json")
	input, err := json.MarshalIndent(content, "", "  ")
	if err != nil {
		return nil, err
	}
	input = append(input, '\n')
	if err := os.WriteFile(inputPath, input, 0o644); err != nil {
		return nil, err
	}

	ffmpegPath := r.FFmpegPath
	if ffmpegPath == "" {
		ffmpegPath, err = exec.LookPath("ffmpeg")
		if err != nil {
			return nil, errors.New("ffmpeg is required for the temporary local video renderer; the Remotion contract was written but MP4 rendering could not run")
		}
	}
	runCommand := r.RunCommand
	if runCommand == nil {
		runCommand = defaultCommandRunner
	}

	outputPath := filepath.Join(outputDir, "ai-news-short.mp4")
	duration := strconv.Itoa(videoDuration(content))
	args := []string{
		"-y",
		"-hide_banner",
		"-loglevel", "error",
		"-f", "lavfi",
		"-i", "color=c=0x172326:s=1080x1920:d=" + duration,
		"-vf", "format=yuv420p",
		"-r", strconv.Itoa(content.FPS),
		"-movflags", "+faststart",
		outputPath,
	}
	if err := runCommand(ctx, ffmpegPath, args...); err != nil {
		return nil, fmt.Errorf("render video with ffmpeg: %w", err)
	}
	if _, err := os.Stat(outputPath); err != nil {
		return nil, fmt.Errorf("expected video output %q: %w", outputPath, err)
	}

	return []OutputFile{
		{Kind: "video_remotion_input", Path: inputPath},
		{Kind: "video_mp4", Path: outputPath},
	}, nil
}

func defaultCommandRunner(ctx context.Context, name string, args ...string) error {
	return exec.CommandContext(ctx, name, args...).Run()
}

func videoDuration(content VideoContent) int {
	total := 0
	for _, scene := range content.Scenes {
		total += scene.DurationSeconds
	}
	if total <= 0 {
		return 3
	}
	return total
}
