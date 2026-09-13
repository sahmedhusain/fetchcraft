package download

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"fetchcraft/src/config"
)


// Downloads files from a list of URLs in the input file asynchronously.
func DownloadMultiple(cfg *config.Config) error {
	file, err := os.Open(cfg.InputFile)
	if err != nil {
		return fmt.Errorf("failed to open input file: %v", err)
	}
	defer file.Close()

	var urls []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			urls = append(urls, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read input file: %v", err)
	}

	if len(urls) == 0 {
		return fmt.Errorf("no URLs found in input file")
	}

	// 1. Retrieve all content sizes
	type urlSize struct {
		url  string
		size int64
		err  error
	}
	sizeChan := make(chan urlSize, len(urls))
	var sizeWg sync.WaitGroup

	for _, u := range urls {
		sizeWg.Add(1)
		go func(targetURL string) {
			defer sizeWg.Done()
			// Use http.Head to get Content-Length quickly
			resp, err := http.Head(targetURL)
			if err != nil {
				sizeChan <- urlSize{url: targetURL, size: -1, err: err}
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK && resp.ContentLength > 0 {
				sizeChan <- urlSize{url: targetURL, size: resp.ContentLength, err: nil}
			} else {
				// Fallback to GET in case HEAD is not supported by target server
				respGet, err := http.Get(targetURL)
				if err != nil {
					sizeChan <- urlSize{url: targetURL, size: -1, err: err}
					return
				}
				defer respGet.Body.Close()
				sizeChan <- urlSize{url: targetURL, size: respGet.ContentLength, err: nil}
			}
		}(u)
	}

	sizeWg.Wait()
	close(sizeChan)

	var sizes []int
	for us := range sizeChan {
		if us.err == nil && us.size >= 0 {
			sizes = append(sizes, int(us.size))
		}
	}
	sort.Ints(sizes)

	sizeStrList := make([]string, len(sizes))
	for i, s := range sizes {
		sizeStrList[i] = fmt.Sprintf("%d", s)
	}
	fmt.Println(styleSizeList(strings.Join(sizeStrList, ", "), cfg.Background))

	// 2. Perform concurrent downloads
	var wg sync.WaitGroup
	var mu sync.Mutex
	downloadedURLs := make([]string, 0, len(urls))
	var firstErr error

	limit, err := ParseRateLimit(cfg.RateLimit)
	if err != nil {
		return err
	}

	for _, u := range urls {
		wg.Add(1)
		go func(targetURL string) {
			defer wg.Done()

			targetPath, _, err := GetDownloadPath(cfg, targetURL)
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("invalid URL %s: %v", targetURL, err)
				}
				mu.Unlock()
				return
			}

			if cfg.OutputDir != "" {
				if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
					mu.Lock()
					if firstErr == nil {
						firstErr = fmt.Errorf("failed to create directory %s: %v", cfg.OutputDir, err)
					}
					mu.Unlock()
					return
				}
			}

			resp, err := http.Get(targetURL)
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("HTTP get failed for %s: %v", targetURL, err)
				}
				mu.Unlock()
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("unsuccessful status %s for %s", resp.Status, targetURL)
				}
				mu.Unlock()
				return
			}

			out, err := os.Create(targetPath)
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("failed to create file %s: %v", targetPath, err)
				}
				mu.Unlock()
				return
			}
			defer out.Close()

			progressReader := NewDownloadProgressReader(resp.Body, resp.ContentLength, limit, true)
			_, err = io.Copy(out, progressReader)
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("download failed for %s: %v", targetURL, err)
				}
				mu.Unlock()
				return
			}

			filename := filepath.Base(targetPath)
			fmt.Println(styleFinished(filename, cfg.Background))

			mu.Lock()
			downloadedURLs = append(downloadedURLs, targetURL)
			mu.Unlock()

		}(u)
	}

	wg.Wait()

	successMap := make(map[string]bool)
	mu.Lock()
	for _, u := range downloadedURLs {
		successMap[u] = true
	}
	mu.Unlock()

	var finalOrdered []string
	for _, u := range urls {
		if successMap[u] {
			finalOrdered = append(finalOrdered, u)
		}
	}

	fmt.Println()
	fmt.Println(styleDownloadFinished(strings.Join(finalOrdered, " "), cfg.Background))

	if firstErr != nil {
		return firstErr
	}
	return nil
}
