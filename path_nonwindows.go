//go:build !windows

package kvega

func fsToURLPath(path string) string {
	return path
}

func urlToFSPath(path string) string {
	return path
}
