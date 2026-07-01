package main

import (
	"fmt"
	"os"
	"os/exec"

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

	if cfg.Background && os.Getenv("WGET_BACKGROUND_CHILD") != "1" {
		logFile, err := os.OpenFile("wget-log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating log file: %v\n", err)
			os.Exit(1)
		}
		defer logFile.Close()

		cmd := exec.Command(os.Args[0], os.Args[1:]...)
		cmd.Env = append(os.Environ(), "WGET_BACKGROUND_CHILD=1")
		cmd.Stdout = logFile
		cmd.Stderr = logFile

		err = cmd.Start()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error starting background process: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Output will be written to ‘wget-log’.")
		os.Exit(0)
	}

	if cfg.InputFile != "" {
		err := download.DownloadMultiple(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if len(cfg.URLs) > 0 {
		err := download.DownloadFile(cfg, cfg.URLs[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
}
