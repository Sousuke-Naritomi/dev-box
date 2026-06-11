package fs

import "os"

// MockFileSystem is an in-memory file system for testing.
type MockFileSystem struct {
	Files map[string][]byte
}

func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		Files: make(map[string][]byte),
	}
}

func (m *MockFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	m.Files[path] = data
	return nil
}

func (m *MockFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return nil
}

func (m *MockFileSystem) Exists(path string) bool {
	_, ok := m.Files[path]
	return ok
}
