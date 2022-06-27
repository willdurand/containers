package container

import "path/filepath"

func GetBaseDir(rootDir string) string {
	return filepath.Join(rootDir, "containers")
}
