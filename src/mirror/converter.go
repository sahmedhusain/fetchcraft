package mirror

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/net/html"
)

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
