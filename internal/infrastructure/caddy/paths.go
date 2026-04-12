package caddy

import (
	"os"
	"path/filepath"
	"strings"
)

// MapHugoverseDataPathToCaddySiteRoot maps a path under Hugoverse's data directory to the path
// Caddy uses for file_server roots when the site tree is bind-mounted at CADDY_SITE_ROOT.
// If dataDir or siteRoot is empty, or hugoversePath is not under dataDir, hugoversePath is returned unchanged
// (same-host CLI, or paths outside the data tree).
func MapHugoverseDataPathToCaddySiteRoot(dataDir, siteRoot, hugoversePath string) string {
	if dataDir == "" || siteRoot == "" || hugoversePath == "" {
		return hugoversePath
	}
	d := filepath.ToSlash(filepath.Clean(dataDir))
	c := filepath.ToSlash(filepath.Clean(siteRoot))
	p := filepath.ToSlash(filepath.Clean(hugoversePath))
	if p == d {
		return c
	}
	prefix := d + "/"
	if strings.HasPrefix(p, prefix) {
		rel := strings.TrimPrefix(p, prefix)
		if rel == "" {
			return c
		}
		return c + "/" + rel
	}
	return hugoversePath
}

// ToCaddySiteRootPath reads HUGOVERSE_DATA_DIR and CADDY_SITE_ROOT from the environment and maps hugoversePath.
func ToCaddySiteRootPath(hugoversePath string) string {
	return MapHugoverseDataPathToCaddySiteRoot(
		os.Getenv("HUGOVERSE_DATA_DIR"),
		os.Getenv("CADDY_SITE_ROOT"),
		hugoversePath,
	)
}
