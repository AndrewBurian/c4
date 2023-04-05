package loader

import "fmt"

type Loader struct {
}

type WorkspaceSource struct {
	File string
	Dsl  []byte
}

func (l *Loader) NextWorkspace() (*WorkspaceSource, error) {
	return nil, fmt.Errorf("unimplemented")
}
