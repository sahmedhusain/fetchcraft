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

// A rate limit string (e.g. "400k", "2M") and returns speed in bytes/sec.
func ParseRateLimit(rateStr string) (int64, error) {
	if rateStr == "" {
		return 0, nil
	}
	rateStr = strings.TrimSpace(rateStr)
	var multiplier int64 = 1
	var numStr string

	if len(rateStr) > 0 {
		lastChar := rateStr[len(rateStr)-1]
		switch lastChar {
		case 'k', 'K':
			multiplier = 1024
			numStr = rateStr[:len(rateStr)-1]
		case 'm', 'M':
			multiplier = 1024 * 1024
			numStr = rateStr[:len(rateStr)-1]
		case 'g', 'G':
			multiplier = 1024 * 1024 * 1024
			numStr = rateStr[:len(rateStr)-1]
		default:
			numStr = rateStr
		}
	}

	var val int64
	_, err := fmt.Sscanf(numStr, "%d", &val)
	if err != nil {
		return 0, fmt.Errorf("invalid rate limit format: %s", rateStr)
	}

	return val * multiplier, nil
}

// Downloads a single URL.
func DownloadFile(cfg *config.Config, rawURL string) error {
	startTime := time.Now()
	fmt.Printf("start at %s\n", startTime.Format("2006-01-02 15:04:05"))

	targetPath, displayPath, err := GetDownloadPath(cfg, rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %v", err)
	}

	limit, err := ParseRateLimit(cfg.RateLimit)
	if err != nil {
		return err
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

	progressReader := NewDownloadProgressReader(resp.Body, resp.ContentLength, limit, cfg.Background)
	_, err = io.Copy(out, progressReader)
	if err != nil {
		return fmt.Errorf("download failed: %v", err)
	}

	if !cfg.Background {
		fmt.Println()
		fmt.Println()
	}

	fmt.Printf("Downloaded [%s]\n", rawURL)
	fmt.Printf("finished at %s\n", time.Now().Format("2006-01-02 15:04:05"))

	return nil
}
