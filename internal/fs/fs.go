package fs

import (
	"os"
	"path/filepath"
)

// FileSystem defines operations for interacting with files.
type FileSystem interface {
	WriteFile(path string, data []byte, perm os.FileMode) error
	MkdirAll(path string, perm os.FileMode) error
	Exists(path string) bool
}

// OSFileSystem implements FileSystem using the standard os package.
type OSFileSystem struct{}

func NewOSFileSystem() *OSFileSystem {
	return &OSFileSystem{}
}

func (fs *OSFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if dir != "." {
		if err := fs.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, data, perm)
}

func (fs *OSFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (fs *OSFileSystem) Exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
