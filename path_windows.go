package kvega

import (
	"path/filepath"
	"strings"
)

func fsToURLPath(path string) string {
	parts := strings.Split(path, string(filepath.Separator))
	urlPath := strings.Join(parts, "/")

	// URL path needs a leading / if a volume name exists.
	if filepath.VolumeName(path) != "" {
		urlPath = "/" + urlPath
	}

	return urlPath
}

func urlToFSPath(path string) string {
	parts := strings.Split(path, "/")
	fsPath := strings.Join(parts, string(filepath.Separator))

	// Strip leading / (now \ since we've replaced / with \)
	// if url path contains volume name.
	if fsPath != "" && filepath.VolumeName(fsPath[1:]) != "" {
		fsPath = fsPath[1:]
	}

	return fsPath
}
