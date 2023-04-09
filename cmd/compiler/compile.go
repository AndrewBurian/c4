package compiler

import (
	"go.burian.dev/c4/cmd/compiler/internal/lexer"
	"go.burian.dev/c4/cmd/compiler/internal/parser"
)

type Compiler interface {
	SourceData(file string) ([]byte, error)
	TokenStream(source string) (lexer.TokenStream, error)
	Model(workspace string) (*parser.Workspace, error)
}
