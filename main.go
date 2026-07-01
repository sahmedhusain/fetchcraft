package main

import (
	"fmt"
	"os"

	"wget/src/config"
	"wget/src/download"
)

func main() {
	cfg, err := config.ParseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing arguments: %v\n", err)
		os.Exit(1)
	}

	if len(cfg.URLs) == 0 && cfg.InputFile == "" {
		fmt.Println("wget: missing URL")
		fmt.Println("Usage: go run . [options] <URL>")
		os.Exit(1)
	}

	if len(cfg.URLs) > 0 {
		err := download.DownloadFile(cfg, cfg.URLs[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
}
