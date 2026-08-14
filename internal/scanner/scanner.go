package scanner

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

func ScanLogDirectory(dirPath string) ([]string, error) {
	var logFiles []string

	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() && strings.HasSuffix(path, ".log") {
			logFiles = append(logFiles, path)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walk directory %s: %w", dirPath, err)
	}

	sort.Strings(logFiles)

	return logFiles, err
}
