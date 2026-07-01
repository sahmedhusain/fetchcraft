# Getting Started Guide 🚀

This guide will walk you through compiling, running, and testing this custom **wget** implementation.

---

## 📋 Table of Contents
1. [First Steps & Prerequisites](#-first-steps--prerequisites)
2. [Compiling the Code](#-compiling-the-code)
3. [Core Features & Usage](#-core-features--usage)
4. [Website Mirroring & Offline Tunnels](#-website-mirroring--offline-tunnels)
5. [Behind the Scenes: How It Works](#-behind-the-scenes-how-it-works)

---

## 🛠 First Steps & Prerequisites

To compile and run this project, you need the Go programming language toolchain installed on your computer. If you do not have Go, you can download it from the official site: [go.dev/dl](https://go.dev/dl/).

Once installed, verify it works by checking your Go version:
```bash
go version
```
*Note: This project is built using modern Go features and requires Go 1.20 or newer.*

---

## 🏗 Compiling the Code

To build the executable binary from the source files:

```bash
# Compiles the source files and outputs a single executable named "wget"
go build -o wget

# Run static analysis and vetting to ensure code matches best practices
go vet ./...
```

Once built, you will see a `wget` (or `wget.exe` on Windows) file in the root directory. You can run all commands directly using `./wget`.

---

## ⚙️ Core Features & Usage

Here are the basic commands for downloading files:

### 1. Downloading to the Current Directory
Downloads the file and preserves its original name.
```bash
./wget https://pbs.twimg.com/media/EMtmPFLWkAA8CIS.jpg
```

### 2. Renaming the Output File (`-O`)
Saves the download under a custom filename.
```bash
./wget -O=meme.jpg https://pbs.twimg.com/media/EMtmPFLWkAA8CIS.jpg
```

### 3. Saving to a Specific Path (`-P`)
Specifies the destination directory. Directories will be created automatically if they do not exist.
```bash
./wget -P=~/Downloads/ -O=meme.jpg https://pbs.twimg.com/media/EMtmPFLWkAA8CIS.jpg
```

### 4. Limiting Download Speeds (`--rate-limit`)
Limits the bandwidth consumption. You can specify limits in bytes/sec, kilobytes/sec (`k` or `K`), or megabytes/sec (`m` or `M`).
```bash
# Limit to 300 Kilobytes per second
./wget --rate-limit=300k https://assets.01-edu.org/wgetDataSamples/20MB.zip

# Limit to 2 Megabytes per second
./wget --rate-limit=2M https://assets.01-edu.org/wgetDataSamples/20MB.zip
```

### 5. Running in the Background (`-B`)
Forks the process into the background, returning terminal control to you immediately. Output details are logged to a file called `wget-log`.
```bash
./wget -B https://assets.01-edu.org/wgetDataSamples/20MB.zip
```

### 6. Concurrent Multi-file Downloads (`-i`)
Downloads multiple files in parallel. The program reads from a file containing a list of URLs (one per line).
```bash
# Create list of files to download
echo -e "https://assets.01-edu.org/wgetDataSamples/Image_10MB.zip\nhttps://assets.01-edu.org/wgetDataSamples/20MB.zip" > download.txt

# Run concurrent batch downloader
./wget -i=download.txt
```

---

## 🌐 Website Mirroring & Offline Tunnels

The mirroring flag (`--mirror`) downloads the website file structure and assets. You can fine-tune what to download using the following optional flags in conjunction with `--mirror`:

| Flag | Meaning | Example |
| :--- | :--- | :--- |
| `-R, --reject` | Suffixes of files to reject and avoid downloading | `--reject=jpg,gif` |
| `-X, --exclude` | Directory path prefixes to exclude from the crawl | `-X=/js,/assets` |
| `--convert-links` | Rewrites links in HTML/CSS to point to local relative files | `--convert-links` |

### Mirroring Example
To mirror the `trypap.com` website, exclude the images folder, and convert links for offline viewing:
```bash
./wget --mirror -X=/img --convert-links https://trypap.com/
```

---

## 🧠 Behind the Scenes: How It Works

### Rate Limiter
The rate limiter calculates how much time *should* have elapsed under the target speed for the number of bytes processed so far:
$$\text{Expected Duration} = \frac{\text{Bytes Transferred}}{\text{Rate Limit}}$$
If the actual elapsed duration is less than the expected duration, the thread sleeps for the difference ($\text{Expected} - \text{Actual}$), creating a highly precise and adaptive speed governor.

### Website Scraper & Crawler
The `--mirror` flag initiates a Breadth-First Search (BFS) crawl:
1. Parses the seed URL to establish the target host.
2. Fetches the page. If it is HTML, it parses the tag tree (for `a`, `link`, and `img` elements) and extracts links. If it is CSS, it scans for `url(...)` declarations using regular expressions.
3. Resolves extracted URLs to absolute paths and validates them against the target host, exclusion paths (`-X`), and file rejections (`-R`).
4. If a URL is valid and unvisited, it is added to the BFS queue and downloaded using the core download module.
5. After crawling finishes, if `--convert-links` is active, the tool loops through all downloaded HTML and CSS files, translating URLs pointing to other downloaded assets into local relative filesystem paths.
