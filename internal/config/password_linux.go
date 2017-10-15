package config

import "fmt"

func loadJIRAPassword(url, username string) (string, error) {
	return "", fmt.Errorf("keychain support not available for Linux yet")
}
