package download

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"wget/src/config"
)

// Formats size in bytes and decimal MB/GB representation.
func FormatContentSize(size int64) string {
	if size < 0 {
		return "unspecified"
	}
	if size >= 1000000000 {
		return fmt.Sprintf("%d [~%.2fGB]", size, float64(size)/1000000000.0)
	}
	return fmt.Sprintf("%d [~%.2fMB]", size, float64(size)/1000000.0)
}

// Determines the target path on disk for saving the file.
func GetDownloadPath(cfg *config.Config, rawURL string) (string, string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", "", err
	}

	filename := cfg.OutputFile
	if filename == "" {
		filename = filepath.Base(parsed.Path)
		if filename == "" || filename == "/" || filename == "." {
			filename = "index.html"
		}
	}

	var targetPath string
	if cfg.OutputDir != "" {
		targetPath = filepath.Join(cfg.OutputDir, filename)
	} else {
		targetPath = filename
	}

	displayPath := targetPath
	if !filepath.IsAbs(displayPath) && !strings.HasPrefix(displayPath, "./") && !strings.HasPrefix(displayPath, "../") {
		displayPath = "./" + displayPath
	}

	return targetPath, displayPath, nil
}

// Downloads a single URL.
func DownloadFile(cfg *config.Config, rawURL string) error {
	startTime := time.Now()
	fmt.Printf("start at %s\n", startTime.Format("2006-01-02 15:04:05"))

	targetPath, displayPath, err := GetDownloadPath(cfg, rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %v", err)
	}

	fmt.Printf("sending request, awaiting response... ")

	resp, err := http.Get(rawURL)
	if err != nil {
		fmt.Println("failed")
		return fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	fmt.Printf("status %s\n", resp.Status)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unsuccessful status: %s", resp.Status)
	}

	fmt.Printf("content size: %s\n", FormatContentSize(resp.ContentLength))
	fmt.Printf("saving file to: %s\n", displayPath)

	// Create directories if OutputDir is specified
	if cfg.OutputDir != "" {
		if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %v", cfg.OutputDir, err)
		}
	}

	out, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %v", targetPath, err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("download failed: %v", err)
	}

	fmt.Printf("\nDownloaded [%s]\n", rawURL)
	fmt.Printf("finished at %s\n", time.Now().Format("2006-01-02 15:04:05"))

	return nil
}
