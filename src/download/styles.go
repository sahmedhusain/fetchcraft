package download

import "strings"

const (
	ColorReset   = "\u001b[0m"
	ColorBold    = "\u001b[1m"
	ColorRed     = "\u001b[31m"
	ColorGreen   = "\u001b[32m"
	ColorYellow  = "\u001b[33m"
	ColorBlue    = "\u001b[34m"
	ColorMagenta = "\u001b[35m"
	ColorCyan    = "\u001b[36m"
	ColorGray    = "\u001b[90m"
)

func styleStart(t string, quiet bool) string {
	if quiet {
		return "start at " + t
	}
	return ColorGray + ColorBold + "start at " + ColorReset + ColorCyan + t + ColorReset
}

func styleEnd(t string, quiet bool) string {
	if quiet {
		return "finished at " + t
	}
	return ColorGray + ColorBold + "finished at " + ColorReset + ColorCyan + t + ColorReset
}

func styleRequest(quiet bool) string {
	if quiet {
		return "sending request, awaiting response... "
	}
	return ColorYellow + "sending request, awaiting response... " + ColorReset
}

func styleStatus(status string, quiet bool) string {
	if quiet {
		return "status " + status
	}
	color := ColorGreen
	if !strings.Contains(status, "200") {
		color = ColorRed
	}
	return ColorBold + "status " + ColorReset + color + ColorBold + status + ColorReset
}

func styleSize(sizeStr string, quiet bool) string {
	if quiet {
		return "content size: " + sizeStr
	}
	return ColorBlue + ColorBold + "content size: " + ColorReset + ColorYellow + sizeStr + ColorReset
}

func styleSaving(path string, quiet bool) string {
	if quiet {
		return "saving file to: " + path
	}
	return ColorBlue + ColorBold + "saving file to: " + ColorReset + ColorCyan + path + ColorReset
}

func styleDownloaded(url string, quiet bool) string {
	if quiet {
		return "Downloaded [" + url + "]"
	}
	return ColorGreen + ColorBold + "Downloaded [" + ColorReset + ColorCyan + url + ColorGreen + ColorBold + "]" + ColorReset
}

func styleSizeList(sizesStr string, quiet bool) string {
	if quiet {
		return "content size: [" + sizesStr + "]"
	}
	return ColorBlue + ColorBold + "content size: " + ColorReset + ColorYellow + "[" + sizesStr + "]" + ColorReset
}

func styleFinished(filename string, quiet bool) string {
	if quiet {
		return "finished " + filename
	}
	return ColorGreen + "finished " + ColorReset + ColorCyan + filename + ColorReset
}

func styleDownloadFinished(finalList string, quiet bool) string {
	if quiet {
		return "Download finished:  [" + finalList + "]"
	}
	return ColorGreen + ColorBold + "Download finished:  [" + ColorReset + ColorCyan + finalList + ColorGreen + ColorBold + "]" + ColorReset
}
