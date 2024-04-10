package utils

import (
	"path/filepath"
	"runtime"
)

func GetCurrentFilePath() string {
	_, filename, _, ok := runtime.Caller(1)
	if !ok {
		panic("Failed to retrieve the file path for Postgres migrations")
	}
	return filename
}

func GetParentDirectory(filePath string) string {
	return filepath.Dir(filePath)
}
