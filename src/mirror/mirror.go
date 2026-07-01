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

// Parses local files (HTML or CSS) and returns referenced URLs.
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
	resolved.Fragment = ""
	return resolved.String()
}

func convertLinksInHTML(filePath string, sourceURL string, visited map[string]string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	doc, err := html.Parse(file)
	file.Close()
	if err != nil {
		return err
	}

	base, err := url.Parse(sourceURL)
	if err != nil {
		return err
	}

	var convertNode func(*html.Node)
	convertNode = func(n *html.Node) {
		if n.Type == html.ElementNode {
			var attrKey string
			switch n.Data {
			case "a", "link":
				attrKey = "href"
			case "img":
				attrKey = "src"
			}

			if attrKey != "" {
				for i, attr := range n.Attr {
					if attr.Key == attrKey {
						val := strings.TrimSpace(attr.Val)
						if val != "" && !strings.HasPrefix(val, "#") && !strings.HasPrefix(val, "javascript:") {
							resolved := resolveURL(val, base)
							if localPath, ok := visited[resolved]; ok {
								relPath, err := filepath.Rel(filepath.Dir(filePath), localPath)
								if err == nil {
									n.Attr[i].Val = filepath.ToSlash(relPath)
								}
							}
						}
					}
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			convertNode(c)
		}
	}

	convertNode(doc)

	outFile, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	return html.Render(outFile, doc)
}

func convertLinksInCSS(filePath string, sourceURL string, visited map[string]string) error {
	contentBytes, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	content := string(contentBytes)

	base, err := url.Parse(sourceURL)
	if err != nil {
		return err
	}

	updatedContent := cssUrlRegex.ReplaceAllStringFunc(content, func(match string) string {
		sub := cssUrlRegex.FindStringSubmatch(match)
		if len(sub) > 1 {
			val := strings.TrimSpace(sub[1])
			if val == "" || strings.HasPrefix(val, "data:") {
				return match
			}
			resolved := resolveURL(val, base)
			if localPath, ok := visited[resolved]; ok {
				relPath, err := filepath.Rel(filepath.Dir(filePath), localPath)
				if err == nil {
					return fmt.Sprintf("url('%s')", filepath.ToSlash(relPath))
				}
			}
		}
		return match
	})

	return os.WriteFile(filePath, []byte(updatedContent), 0644)
}
