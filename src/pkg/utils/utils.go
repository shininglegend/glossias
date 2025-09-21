package utils

import "strings"

func SanitizeFileName(name string) string {
	sanitizedFilename := strings.ReplaceAll(name, "/", "")
	sanitizedFilename = strings.ReplaceAll(sanitizedFilename, "\\", "")
	sanitizedFilename = strings.ReplaceAll(sanitizedFilename, "..", "")
	return sanitizedFilename
}
