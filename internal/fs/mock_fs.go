package fs

import (
	"fmt"
	"os"
)

// MockFile represents a file in the MockFileSystem
type MockFile struct {
	Data []byte
	Perm os.FileMode
}

// MockFileSystem is an in-memory file system for testing.
type MockFileSystem struct {
	Files    map[string]MockFile
	Symlinks map[string]bool
}

func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		Files:    make(map[string]MockFile),
		Symlinks: make(map[string]bool),
	}
}

func (m *MockFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	if m.Symlinks[path] {
		return fmt.Errorf("refusing to overwrite symbolic link: %s", path)
	}
	m.Files[path] = MockFile{Data: data, Perm: perm}
	return nil
}

func (m *MockFileSystem) WriteFiles(files map[string]File) error {
	for path := range files {
		if m.Symlinks[path] {
			return fmt.Errorf("refusing to overwrite symbolic link: %s", path)
		}
	}
	for path, file := range files {
		m.Files[path] = MockFile{Data: append([]byte(nil), file.Data...), Perm: file.Perm}
	}
	return nil
}

func (m *MockFileSystem) ReadFile(path string) ([]byte, error) {
	file, ok := m.Files[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return append([]byte(nil), file.Data...), nil
}

func (m *MockFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return nil
}

func (m *MockFileSystem) Exists(path string) bool {
	_, ok := m.Files[path]
	return ok
}

func (m *MockFileSystem) IsSymlink(path string) (bool, error) {
	return m.Symlinks[path], nil
}
