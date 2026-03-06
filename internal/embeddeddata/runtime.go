package embeddeddata

import (
	"fmt"
	"os"
	"path/filepath"
)

func DefaultCacheDir() (string, error) {
	preferred, err := os.UserCacheDir()
	if err != nil {
		preferred = ""
	}
	fallback := os.TempDir()
	if preferred == "" && fallback == "" {
		return "", fmt.Errorf("resolve user cache dir")
	}
	preferredPath := filepath.Join(preferred, "pipescope", "embeddeddata")
	fallbackPath := filepath.Join(fallback, "pipescope", "embeddeddata")
	return resolveWritableCacheDir(preferredPath, fallbackPath)
}

func resolveWritableCacheDir(preferredPath, fallbackPath string) (string, error) {
	for _, path := range []string{preferredPath, fallbackPath} {
		if path == "" {
			continue
		}
		if err := os.MkdirAll(path, 0o755); err != nil {
			continue
		}
		probe := filepath.Join(path, ".probe")
		if err := os.WriteFile(probe, []byte("ok"), 0o644); err != nil {
			continue
		}
		_ = os.Remove(probe)
		return path, nil
	}
	return "", fmt.Errorf("resolve writable cache dir")
}
