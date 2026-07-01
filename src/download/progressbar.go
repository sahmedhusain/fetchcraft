package download

import (
	"fmt"
	"io"
	"strings"
	"time"
)

// Formats bytes into KiB, MiB, or GiB.
func FormatBytesBinary(bytes float64) string {
	if bytes >= 1024*1024*1024 {
		return fmt.Sprintf("%.2f GiB", bytes/(1024*1024*1024))
	}
	if bytes >= 1024*1024 {
		return fmt.Sprintf("%.2f MiB", bytes/(1024*1024))
	}
	return fmt.Sprintf("%.2f KiB", bytes/1024)
}

// Formats transfer speed in KiB/s, MiB/s, or GiB/s.
func FormatSpeed(bytesPerSec float64) string {
	if bytesPerSec >= 1024*1024*1024 {
		return fmt.Sprintf("%.2f GiB/s", bytesPerSec/(1024*1024*1024))
	}
	if bytesPerSec >= 1024*1024 {
		return fmt.Sprintf("%.2f MiB/s", bytesPerSec/(1024*1024))
	}
	if bytesPerSec >= 1024 {
		return fmt.Sprintf("%.2f KiB/s", bytesPerSec/1024)
	}
	return fmt.Sprintf("%.2f B/s", bytesPerSec)
}

// Formats remaining time to format: e.g. 0s, 45s, 1m2s, 1h2m3s.
func FormatETA(duration time.Duration) string {
	duration = duration.Round(time.Second)
	if duration < 0 {
		return "0s"
	}
	h := duration / time.Hour
	duration -= h * time.Hour
	m := duration / time.Minute
	duration -= m * time.Minute
	s := duration / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh%dm%ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm%ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

type DownloadProgressReader struct {
	reader     io.Reader
	totalSize  int64
	downloaded int64
	startTime  time.Time
	rateLimit  int64 // bytes/sec
	lastUpdate time.Time
	quiet      bool
}

func NewDownloadProgressReader(r io.Reader, totalSize int64, rateLimit int64, quiet bool) *DownloadProgressReader {
	return &DownloadProgressReader{
		reader:     r,
		totalSize:  totalSize,
		startTime:  time.Now(),
		rateLimit:  rateLimit,
		lastUpdate: time.Now(),
		quiet:      quiet,
	}
}

func (pr *DownloadProgressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	if n > 0 {
		pr.downloaded += int64(n)

		// Enforce rate limit
		if pr.rateLimit > 0 {
			now := time.Now()
			elapsed := now.Sub(pr.startTime)
			expected := time.Duration(float64(pr.downloaded) / float64(pr.rateLimit) * float64(time.Second))
			if elapsed < expected {
				time.Sleep(expected - elapsed)
			}
		}

		// Update progress bar
		now := time.Now()
		if !pr.quiet && (now.Sub(pr.lastUpdate) >= 50*time.Millisecond || err != nil || pr.downloaded == pr.totalSize) {
			pr.drawProgressBar()
			pr.lastUpdate = now
		}
	}
	return n, err
}

func (pr *DownloadProgressReader) drawProgressBar() {
	elapsed := time.Since(pr.startTime).Seconds()
	if elapsed <= 0 {
		elapsed = 0.001
	}
	speed := float64(pr.downloaded) / elapsed

	if pr.totalSize > 0 {
		pct := (float64(pr.downloaded) / float64(pr.totalSize)) * 100.0
		if pct > 100.0 {
			pct = 100.0
		}

		// Render bar
		barWidth := 50
		completedWidth := int((pct / 100.0) * float64(barWidth))
		if completedWidth > barWidth {
			completedWidth = barWidth
		}

		bar := strings.Repeat("=", completedWidth)
		if completedWidth < barWidth {
			bar += strings.Repeat(" ", barWidth-completedWidth)
		}

		var etaStr string
		if speed > 0 {
			eta := time.Duration(float64(pr.totalSize-pr.downloaded)/speed) * time.Second
			etaStr = FormatETA(eta)
		} else {
			etaStr = "--"
		}

		fmt.Printf("\r %s / %s [%s] %6.2f%% %s %s",
			FormatBytesBinary(float64(pr.downloaded)),
			FormatBytesBinary(float64(pr.totalSize)),
			bar,
			pct,
			FormatSpeed(speed),
			etaStr,
		)
	} else {
		// Unknown size progress
		fmt.Printf("\r %s [   <=>   ] %s",
			FormatBytesBinary(float64(pr.downloaded)),
			FormatSpeed(speed),
		)
	}
}
