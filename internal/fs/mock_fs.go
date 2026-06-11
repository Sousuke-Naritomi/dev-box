package fs

import "os"

// MockFile represents a file in the MockFileSystem
type MockFile struct {
	Data []byte
	Perm os.FileMode
}

// MockFileSystem is an in-memory file system for testing.
type MockFileSystem struct {
	Files map[string]MockFile
}

func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		Files: make(map[string]MockFile),
	}
}

func (m *MockFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	m.Files[path] = MockFile{Data: data, Perm: perm}
	return nil
}

func (m *MockFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return nil
}

func (m *MockFileSystem) Exists(path string) bool {
	_, ok := m.Files[path]
	return ok
}
