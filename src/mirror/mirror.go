package mirror

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"wget/src/config"
	"wget/src/download"

	"golang.org/x/net/html"
)

var cssUrlRegex = regexp.MustCompile(`url\s*\(\s*['"]?([^'")\s]+)['"]?\s*\)`)

// Starts the website mirroring process.
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

	// Crawl state
	type crawlItem struct {
		url   string
		depth int
	}

	queue := []crawlItem{{url: seedURL, depth: 0}}
	visited := make(map[string]string) // Maps absolute URL to local relative path
	maxDepth := 5

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

	return nil
}

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

func extractLinks(filePath string, sourceURL string, parsedSource *url.URL) []string {
	file, err := os.Open(filePath)
	if err != nil {
		return nil
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(filePath))
	var links []string

	if ext == ".html" || ext == ".htm" {
		links = extractFromHTML(file, sourceURL, parsedSource)
	} else if ext == ".css" {
		links = extractFromCSS(file, sourceURL, parsedSource)
	}

	return links
}

func extractFromHTML(r io.Reader, sourceURL string, parsedSource *url.URL) []string {
	var links []string
	tokenizer := html.NewTokenizer(r)

	for {
		tokenType := tokenizer.Next()
		if tokenType == html.ErrorToken {
			break
		}

		token := tokenizer.Token()
		var attrName string

		switch token.Data {
		case "a", "link":
			attrName = "href"
		case "img":
			attrName = "src"
		case "style":
			if tokenType == html.StartTagToken {
				tokenizer.Next()
				innerToken := tokenizer.Token()
				if innerToken.Type == html.TextToken {
					links = append(links, extractCSSUrls(innerToken.Data, parsedSource)...)
				}
			}
			continue
		default:
			for _, attr := range token.Attr {
				if attr.Key == "style" {
					links = append(links, extractCSSUrls(attr.Val, parsedSource)...)
				}
			}
			continue
		}

		for _, attr := range token.Attr {
			if attr.Key == attrName {
				val := strings.TrimSpace(attr.Val)
				if val == "" || strings.HasPrefix(val, "#") || strings.HasPrefix(val, "javascript:") {
					continue
				}
				resolved := resolveURL(val, parsedSource)
				if resolved != "" {
					links = append(links, resolved)
				}
			}
		}
	}

	return links
}

func extractFromCSS(r io.Reader, sourceURL string, parsedSource *url.URL) []string {
	content, err := io.ReadAll(r)
	if err != nil {
		return nil
	}
	return extractCSSUrls(string(content), parsedSource)
}

func extractCSSUrls(css string, parsedSource *url.URL) []string {
	var links []string
	matches := cssUrlRegex.FindAllStringSubmatch(css, -1)
	for _, m := range matches {
		if len(m) > 1 {
			val := strings.TrimSpace(m[1])
			if val == "" || strings.HasPrefix(val, "data:") {
				continue
			}
			resolved := resolveURL(val, parsedSource)
			if resolved != "" {
				links = append(links, resolved)
			}
		}
	}
	return links
}

func resolveURL(ref string, base *url.URL) string {
	refURL, err := url.Parse(ref)
	if err != nil {
		return ""
	}
	resolved := base.ResolveReference(refURL)
	// Strip fragment
	resolved.Fragment = ""
	return resolved.String()
}
