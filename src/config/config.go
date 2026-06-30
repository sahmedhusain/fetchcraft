package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	URLs         []string
	OutputFile   string
	OutputDir    string
	Background   bool
	RateLimit    string
	InputFile    string
	Mirror       bool
	Reject       string
	Exclude      string
	ConvertLinks bool
}

func ParseArgs(args []string) (*Config, error) {
	cfg := &Config{}

	for i := 0; i < len(args); i++ {
		arg := args[i]

		if strings.HasPrefix(arg, "-") {
			var key, val string
			hasEquals := false
			if idx := strings.Index(arg, "="); idx != -1 {
				key = arg[:idx]
				val = arg[idx+1:]
				hasEquals = true
			} else {
				key = arg
			}

			switch key {
			case "-O":
				if hasEquals {
					cfg.OutputFile = val
				} else {
					if i+1 >= len(args) {
						return nil, fmt.Errorf("missing value for option -O")
					}
					cfg.OutputFile = args[i+1]
					i++
				}
			case "-P":
				var dir string
				if hasEquals {
					dir = val
				} else {
					if i+1 >= len(args) {
						return nil, fmt.Errorf("missing value for option -P")
					}
					dir = args[i+1]
					i++
				}
				// Resolve home directory path if it starts with ~
				resolved, err := ExpandHomeDir(dir)
				if err != nil {
					return nil, err
				}
				cfg.OutputDir = resolved
			case "-B":
				cfg.Background = true
			case "--rate-limit":
				if hasEquals {
					cfg.RateLimit = val
				} else {
					if i+1 >= len(args) {
						return nil, fmt.Errorf("missing value for option --rate-limit")
					}
					cfg.RateLimit = args[i+1]
					i++
				}
			case "-i":
				if hasEquals {
					cfg.InputFile = val
				} else {
					if i+1 >= len(args) {
						return nil, fmt.Errorf("missing value for option -i")
					}
					cfg.InputFile = args[i+1]
					i++
				}
			case "--mirror":
				cfg.Mirror = true
			case "-R", "--reject":
				if hasEquals {
					cfg.Reject = val
				} else {
					if i+1 >= len(args) {
						return nil, fmt.Errorf("missing value for option %s", key)
					}
					cfg.Reject = args[i+1]
					i++
				}
			case "-X", "--exclude":
				if hasEquals {
					cfg.Exclude = val
				} else {
					if i+1 >= len(args) {
						return nil, fmt.Errorf("missing value for option %s", key)
					}
					cfg.Exclude = args[i+1]
					i++
				}
			case "--convert-links":
				cfg.ConvertLinks = true
			default:
				return nil, fmt.Errorf("unknown flag: %s", key)
			}
		} else {
			cfg.URLs = append(cfg.URLs, arg)
		}
	}

	return cfg, nil
}

// Expands the leading "~" in path to the user's home directory.
func ExpandHomeDir(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return home, nil
	}
	if strings.HasPrefix(path, "~"+string(filepath.Separator)) || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}
