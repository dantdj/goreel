package video

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
)

// Given an ID and the video data, creates an m3u8 playlist.
// Returns a list of strings containing the filenames
func GenerateM3U8(videoId, videoPath, baseDir string) error {
	hlsPlaylistName := "playlist.m3u8"
	hlsSegmentName := "segment_%03d.ts" // FFMpeg will replace %03d with a number

	var args = []string{
		"-i", videoPath, // Input file
		"-g", "60", // Split keyframes every 60 frames
		"-codec:v", "h264", // Video codec
		"-preset", "veryfast", // Encoding preset (balance speed/quality)
		"-b:v", "1M", // Video bitrate (e.g., 1 Mbps)
		"-maxrate", "1.2M", // Max video bitrate
		"-bufsize", "1.8M", // Buffer size
		"-vf", "scale=-2:720", // Scale to 720p, maintain aspect ratio
		"-codec:a", "aac", // Audio codec
		"-b:a", "128k", // Audio bitrate
		"-f", "hls", // Output format HLS
		"-hls_time", "2", // Segment duration in seconds
		"-hls_playlist_type", "vod", // VOD for on-demand playback
		"-hls_segment_filename", filepath.Join(baseDir, hlsSegmentName), // Path for segments
		filepath.Join(baseDir, hlsPlaylistName), // Path for the main HLS playlist
	}

	slog.Info("Running FFmpeg with args", slog.String("args", fmt.Sprintf("%v", args)))

	cmd := exec.Command("ffmpeg", args...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		slog.Error("FFmpeg failed", slog.String("error", err.Error()))
		return err
	}

	return nil
}
