package generator

import "os"

// FileDef holds the content and permissions for a file to be generated.
type FileDef struct {
	Content string
	Perm    os.FileMode
}
