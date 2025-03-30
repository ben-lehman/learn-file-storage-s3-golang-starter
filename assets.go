package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func (cfg apiConfig) ensureAssetsDir() error {
	if _, err := os.Stat(cfg.assetsRoot); os.IsNotExist(err) {
		return os.Mkdir(cfg.assetsRoot, 0755)
	}
	return nil
}

func getAssetPath(fileName, mediaType string) string {
	ext := mediaTypeToExt(mediaType)
	return fmt.Sprintf("%s%s", fileName, ext)
}

func (cfg apiConfig) getAssetDiskPath(assetPath string) string {
	return filepath.Join(cfg.assetsRoot, assetPath)
}

func (cfg apiConfig) getAssetURL(assetPath string) string {
	return fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, assetPath)
}

func mediaTypeToExt(mediaType string) string {
	parts := strings.Split(mediaType, "/")
	if len(parts) != 2 {
		return ".bin"
	}
	return "." + parts[1]
}

func getVideoAspectRatio(filePath string) (string, error) {
	type videoInfo struct {
		Streams []struct {
			DisplayAspectRatio string `json:"display_aspect_ratio"`
		} `json:"streams"`
	}
	log.Println("getting aspect ratio...")
	cmd := exec.Command("ffprobe", "-v", "error", "-of", "json", "-show_streams", filePath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("ffprobe failed: %v\nOutput: %s", err, output)
		return "", err
	}
	var data videoInfo
	err = json.Unmarshal(output, &data)
	if err != nil {
		log.Println("issue with unmarshaling output")
		return "", err
	}

	log.Printf("video data: %v", data)

	aspectRatio := data.Streams[0].DisplayAspectRatio
	if aspectRatio == "16:9" {
		return "landscape", nil
	}

	if aspectRatio == "9:16" {
		return "portrait", nil
	}

	return "other", nil
}

func processVideoForFastStart(filePath string) (string, error) {
  outputFilePath := filePath + ".processing"

  cmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", outputFilePath)
  output, err := cmd.CombinedOutput()
  if err != nil {
    log.Printf("ffmpeg failed: %v\nOutput: %s", err, output)
    return "", err
  }

  return outputFilePath, nil
}


