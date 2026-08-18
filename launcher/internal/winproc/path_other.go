//go:build !windows

package winproc

// ShortPath returns the path unchanged: only Windows has a short form.
func ShortPath(path string) string { return path }
