package mirror

import (
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

var cssUrlRegex = regexp.MustCompile(`url\s*\(\s*['"]?([^'")\s]+)['"]?\s*\)`)

// extractLinks parses local files (HTML or CSS) and returns referenced URLs.
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
