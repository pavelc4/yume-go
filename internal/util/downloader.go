package util

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type DownloadResult struct {
	FilePath   string
	FolderPath string
	FileSize   int64
}

func DownloadToTemp(url string, identifier string) (*DownloadResult, error) {
	tmpBase := os.TempDir()
	folderName := fmt.Sprintf("waifu_%s_%d", identifier, time.Now().Unix())
	folderPath := filepath.Join(tmpBase, folderName)

	if err := os.MkdirAll(folderPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp folder: %w", err)
	}

	log.Printf("Created temp folder: %s", folderPath)

	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		os.RemoveAll(folderPath)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "image/webp,image/apng,image/*,*/*;q=0.8")
	req.Header.Set("Referer", "https://waifu.im/")

	resp, err := client.Do(req)
	if err != nil {
		os.RemoveAll(folderPath)
		return nil, fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		os.RemoveAll(folderPath)
		return nil, fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	ext := getImageExtension(resp.Header.Get("Content-Type"))
	filename := fmt.Sprintf("waifu_%s%s", identifier, ext)
	filePath := filepath.Join(folderPath, filename)

	out, err := os.Create(filePath)
	if err != nil {
		os.RemoveAll(folderPath)
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	maxSize := int64(50 * 1024 * 1024)
	limitedReader := io.LimitReader(resp.Body, maxSize)
	written, err := io.Copy(out, limitedReader)
	if err != nil {
		os.RemoveAll(folderPath)
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	if written < 512 {
		os.RemoveAll(folderPath)
		return nil, fmt.Errorf("file too small: %d bytes", written)
	}

	if written >= maxSize {
		log.Printf("Warning: file truncated at %d bytes (hit 50MB limit)", written)
	}

	log.Printf("Downloaded: %s (%.2f MB)", filePath, float64(written)/(1024*1024))

	return &DownloadResult{
		FilePath:   filePath,
		FolderPath: folderPath,
		FileSize:   written,
	}, nil
}

func CleanupTemp(folderPath string) error {
	if err := os.RemoveAll(folderPath); err != nil {
		return fmt.Errorf("failed to cleanup: %w", err)
	}
	log.Printf("Cleaned up: %s", folderPath)
	return nil
}

var imageExt = map[string]string{
	"image/jpeg": ".jpg",
	"image/jpg":  ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

func getImageExtension(contentType string) string {
	if ext, ok := imageExt[contentType]; ok {
		return ext
	}
	return ".jpg"
}
