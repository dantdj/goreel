package video

import (
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/dantdj/goreel/storage"
)

type Processor struct {
	Storage storage.Service
}

func NewProcessor(s storage.Service) *Processor {
	return &Processor{
		Storage: s,
	}
}

func (p *Processor) Process(videoId string) error {
	slog.Info("Starting video processing", slog.String("video_id", videoId))

	baseDir := filepath.Join(os.TempDir(), videoId)
	inputDir := filepath.Join(baseDir, "input")
	inputPath := filepath.Join(inputDir, videoId)

	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return fmt.Errorf("failed to make temp directory %s: %w", baseDir, err)
	}
	defer p.cleanup(baseDir)

	if err := p.downloadVideo(videoId, inputDir); err != nil {
		return err
	}
	slog.Info("Video downloaded to temp", slog.String("video_id", videoId))

	// Transcode
	if err := p.generateM3U8(videoId, inputPath, baseDir); err != nil {
		return fmt.Errorf("failed to generate M3U8 playlist: %w", err)
	}
	slog.Info("HLS generation complete", slog.String("video_id", videoId))

	playlistFiles, err := p.getFilePaths(baseDir)
	if err != nil {
		return fmt.Errorf("failed to get file paths: %w", err)
	}

	slog.Info("Uploading segments", slog.String("video_id", videoId), slog.Int("count", len(playlistFiles)))

	for _, path := range playlistFiles {
		if err := p.uploadFile(path); err != nil {
			return fmt.Errorf("failed to upload file %s: %w", path, err)
		}
	}

	// Delete the local files to avoid cluttering up the temp directory
	if err := p.Storage.Delete(videoId); err != nil {
		return fmt.Errorf("failed to delete video from storage: %w", err)
	}

	slog.Info("Video processing complete", slog.String("video_id", videoId))

	return nil
}

func (p *Processor) downloadVideo(videoId, inputDir string) error {
	videoData, _, _ := p.Storage.Retrieve(videoId)
	defer videoData.Close()

	if err := os.MkdirAll(inputDir, 0755); err != nil {
		return fmt.Errorf("failed to make input directory %s: %w", inputDir, err)
	}

	filepath := filepath.Join(inputDir, videoId)
	outputFile, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create temp input file %s: %w", filepath, err)
	}
	defer outputFile.Close()

	_, err = io.Copy(outputFile, videoData)
	if err != nil {
		return fmt.Errorf("failed to copy video data to temp file: %w", err)
	}
	return nil
}

func (p *Processor) uploadFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// The file name is used as the blob name
	p.Storage.Upload(file, filepath.Base(path))
	return nil
}

func (p *Processor) cleanup(dir string) {
	if err := os.RemoveAll(dir); err != nil {
		slog.Error("Failed to delete temp files", slog.String("error", err.Error()))
	}
}

// getFilePaths returns a slice of file paths within a given directory.
func (p *Processor) getFilePaths(dirPath string) ([]string, error) {
	var filePaths []string

	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip the input directory, we only want the output files
		if d.IsDir() && path == filepath.Join(dirPath, "input") {
			return filepath.SkipDir
		}

		if !d.IsDir() { // Only add files, not directories
			filePaths = append(filePaths, path)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking directory: %w", err)
	}

	return filePaths, nil
}

// Given an ID and the video data, creates an m3u8 playlist.
// Returns a list of strings containing the filenames
func (p *Processor) generateM3U8(videoId, videoPath, baseDir string) error {
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

	// Capture combined output for logging on error
	output, err := cmd.CombinedOutput()
	if err != nil {
		slog.Error("FFmpeg failed", slog.String("output", string(output)), slog.String("error", err.Error()))
		return fmt.Errorf("FFmpeg failed: %w", err)
	}

	return nil
}
