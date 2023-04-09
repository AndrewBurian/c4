package parser

import (
	"bytes"
	"fmt"
	"path"
	"strings"

	"go.burian.dev/c4/cmd/compiler/internal/lexer"
)

type Parser struct {
	code *bytes.Reader

	currentToken  *lexer.Token
	previousToken *lexer.Token

	currentScope    []string
	currentGroup    string
	heldIds         []IdentifierString
	currentFile     string
	currentUniqueId int

	provider Provider

	tokenStreamStack   []*streamState
	currentTokenStream lexer.TokenStream
}

type streamState struct {
	stream   lexer.TokenStream
	previous *lexer.Token
}

type Provider interface {
	GetSourceFor(string) (*bytes.Reader, error)
	GetTokenStreamFor(string) (lexer.TokenStream, error)
}

func (p *Parser) Run(target string, deps Provider) (*Workspace, error) {

	p.provider = deps

	tokens, err := deps.GetTokenStreamFor(target)
	if err != nil {
		return nil, err
	}
	p.currentTokenStream = tokens

	data, err := deps.GetSourceFor(target)
	if err != nil {
		return nil, err
	}
	p.code = data
	p.currentFile = target

	p.currentToken = &lexer.Token{}

	return p.runParse()
}

func (p *Parser) currentSymbol() string {
	symbolLen := p.currentToken.Positions().End.ByteOffset - p.currentToken.Positions().Start.ByteOffset
	symbolBytes := make([]byte, symbolLen)

	targetSourceFile := p.currentToken.Positions().Start.File
	if p.currentFile != targetSourceFile {
		newCode, err := p.provider.GetSourceFor(targetSourceFile)
		if err != nil {
			panic("unable to get source for token: " + err.Error())
		}
		p.currentFile = targetSourceFile
		p.code = newCode
	}
	n, err := p.code.ReadAt(symbolBytes, int64(p.currentToken.Positions().Start.ByteOffset))
	if err != nil || n != symbolLen {
		panic("failed to read data bytes for code")
	}
	return string(symbolBytes)
}

// returns the next token in the stream
//
// There are two special cases
// - the next token is a #pragma
// - the next token is EOF
//
// If it's a pragma, the parser will handle the pragma then advance and return the next token.
// If it's EOF, the parser will check there are streams on the stack. If there are, it will pop
// one and continue that stream, otherwise return EOF
func (p *Parser) nextToken() *lexer.Token {
	p.previousToken = p.currentToken
	p.currentToken = p.currentTokenStream.NextToken()

	// pragma directives are trapped by the parser
	// and not returned to the model
	if p.currentToken.Is(lexer.TypePragma) {
		switch p.currentSymbol() {
		case "#include":
			oldPrev := p.previousToken
			file, err := p.parseString()
			if err != nil {
				panic("failed to process #include pragma: need file argument")
			}

			// load the new inlcuded stream
			newStream, err := p.provider.GetTokenStreamFor(file)
			if err != nil {
				panic("could not fetch named token stream to include: " + err.Error())
			}

			newNext := newStream.NextToken()

			// push the current token source onto the stack
			ss := &streamState{
				stream:   p.currentTokenStream,
				previous: oldPrev,
			}
			p.tokenStreamStack = append(p.tokenStreamStack, ss)

			p.currentTokenStream = newStream
			p.currentToken = newNext
			return p.currentToken
		}
	}

	for p.currentToken.Is(lexer.TypeEOF) && len(p.tokenStreamStack) > 0 {
		popState := p.tokenStreamStack[len(p.tokenStreamStack)-1]
		p.currentTokenStream = popState.stream
		p.currentToken = p.currentTokenStream.NextToken()
		p.previousToken = popState.previous
		p.tokenStreamStack = p.tokenStreamStack[:len(p.tokenStreamStack)-1]
	}

	return p.currentToken
}

func (p *Parser) backupToken() {
	if p.previousToken == nil {
		panic("attempt to double backup tokens")
	}
	p.currentToken = p.previousToken
	p.previousToken = nil
	p.currentTokenStream.BackupToken()
}

func (p *Parser) acceptOne(t lexer.TokenType) bool {
	n := p.nextToken()
	if !n.Is(t) {
		p.backupToken()
		return false
	}
	return true
}

func (p *Parser) holdIdentifierForAssignment(id IdentifierString) {
	p.heldIds = append(p.heldIds, id)
}

func (p *Parser) claimHeldIdentifier() IdentifierString {
	if len(p.heldIds) == 0 {
		panic("attempt to claim null identifier")
	}
	id := p.heldIds[len(p.heldIds)-1]
	p.heldIds = p.heldIds[:len(p.heldIds)]
	return id
}

func (p *Parser) assignIdentifier(e Entity) {
	if len(p.heldIds) > 0 {
		id := p.heldIds[len(p.heldIds)-1]
		p.heldIds = p.heldIds[:len(p.heldIds)-1]
		e.SetId(id)
		return
	}

	typeName := ""
	fallback := ""
	switch obj := e.(type) {
	case *Workspace:
		typeName = "workspace"
		fallback = p.currentFile
	case *SoftwareSystem:
		typeName = "softwaresystem"
		fallback = obj.Name
	case *Container:
		typeName = "container"
		fallback = obj.Name
	case *Component:
		typeName = "component"
		fallback = obj.Name
	}
	id := fmt.Sprintf("_%s%02d_%s", typeName, p.currentUniqueId,
		strings.ToLower(
			strings.TrimSpace(
				strings.ReplaceAll(fallback, " ", "_"),
			),
		),
	)
	p.currentUniqueId++
	e.SetId(IdentifierString(id))
}

func (p *Parser) enterGroup(name string) {
	p.currentGroup = name
}

func (p *Parser) leaveGroup() {
	p.currentGroup = ""
}

func (p *Parser) assignGroup(e Entity) {
	if p.currentGroup != "" {
		e.SetGroup(p.currentGroup)
	}
}

func (p *Parser) workspaceNameFromFile() IdentifierString {
	return IdentifierString(path.Base(p.currentFile))
}
