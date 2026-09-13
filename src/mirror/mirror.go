package mirror

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"fetchcraft/src/config"
	"fetchcraft/src/download"
)

// MirrorSite starts the website mirroring process.
func MirrorSite(cfg *config.Config, seedURL string) error {
	parsedSeed, err := url.Parse(seedURL)
	if err != nil {
		return fmt.Errorf("invalid seed URL: %v", err)
	}

	targetHost := parsedSeed.Host
	if targetHost == "" {
		return fmt.Errorf("URL must have a valid host")
	}

	// Base directory name matches the host (e.g. trypap.com)
	baseDir := targetHost
	err = os.MkdirAll(baseDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create base directory %s: %v", baseDir, err)
	}

	var rejects []string
	if cfg.Reject != "" {
		rejects = strings.Split(cfg.Reject, ",")
	}

	var excludes []string
	if cfg.Exclude != "" {
		excludes = strings.Split(cfg.Exclude, ",")
	}

	// Crawl state
	type crawlItem struct {
		url   string
		depth int
	}

	queue := []crawlItem{{url: seedURL, depth: 0}}
	visited := make(map[string]string) // Maps absolute URL to local relative path
	maxDepth := 5                      // reasonable default depth

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]

		if _, exists := visited[item.url]; exists {
			continue
		}

		parsedItem, err := url.Parse(item.url)
		if err != nil {
			continue
		}

		// Ensure we stay on the same host
		if parsedItem.Host != targetHost {
			continue
		}

		// Exclude check
		if isExcluded(parsedItem, excludes) {
			continue
		}

		// Reject check
		if isRejected(parsedItem, rejects) {
			continue
		}

		// Determine local directory and filename
		_, localDir, filename := getLocalPathDetails(parsedItem, baseDir)

		// Download the file using our download engine
		fileCfg := *cfg
		fileCfg.OutputDir = localDir
		fileCfg.OutputFile = filename

		err = download.DownloadFile(&fileCfg, item.url)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to download %s: %v\n", item.url, err)
			continue
		}

		// Record successfully downloaded file path
		localFullPath := filepath.Join(localDir, filename)
		visited[item.url] = localFullPath

		// If depth limit reached, do not enqueue more links
		if item.depth >= maxDepth {
			continue
		}

		extracted := extractLinks(localFullPath, item.url, parsedItem)
		for _, nextURL := range extracted {
			if _, seen := visited[nextURL]; !seen {
				queue = append(queue, crawlItem{url: nextURL, depth: item.depth + 1})
			}
		}
	}

	if cfg.ConvertLinks {
		for uStr, localPath := range visited {
			ext := strings.ToLower(filepath.Ext(localPath))
			if ext == ".html" || ext == ".htm" {
				_ = convertLinksInHTML(localPath, uStr, visited)
			} else if ext == ".css" {
				_ = convertLinksInCSS(localPath, uStr, visited)
			}
		}
	}

	return nil
}

// getLocalPathDetails computes the target directory and filename for a URL.
func getLocalPathDetails(u *url.URL, baseDir string) (relPath string, dir string, filename string) {
	path := u.Path
	if path == "" || path == "/" {
		dir = baseDir
		filename = "index.html"
		relPath = filepath.Join(dir, filename)
		return
	}

	cleanPath := filepath.Clean(path)
	if strings.HasSuffix(path, "/") {
		dir = filepath.Join(baseDir, cleanPath)
		filename = "index.html"
	} else if filepath.Ext(cleanPath) == "" {
		dir = filepath.Join(baseDir, cleanPath)
		filename = "index.html"
	} else {
		dir = filepath.Join(baseDir, filepath.Dir(cleanPath))
		filename = filepath.Base(cleanPath)
	}

	relPath = filepath.Join(dir, filename)
	return
}

// Checks if URL path matches any of the exclude prefixes.
func isExcluded(u *url.URL, excludes []string) bool {
	if len(excludes) == 0 {
		return false
	}
	path := u.Path
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	for _, x := range excludes {
		x = strings.TrimSpace(x)
		if x == "" {
			continue
		}
		if !strings.HasPrefix(x, "/") {
			x = "/" + x
		}
		if strings.HasPrefix(path, x) {
			return true
		}
	}
	return false
}

// Checks if filename suffix matches any of the reject suffixes.
func isRejected(u *url.URL, rejects []string) bool {
	if len(rejects) == 0 {
		return false
	}
	path := strings.ToLower(u.Path)
	for _, r := range rejects {
		r = strings.TrimSpace(strings.ToLower(r))
		if r == "" {
			continue
		}
		dotSuffix := "." + r
		if strings.HasSuffix(path, dotSuffix) || strings.HasSuffix(path, r) {
			return true
		}
	}
	return false
}
