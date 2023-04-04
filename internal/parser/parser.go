package parser

import (
	"fmt"
	"path"
	"strings"

	"go.burian.dev/c4arch/internal/lexer"
	"go.burian.dev/c4arch/internal/loader"
)

type Parser struct {
	code []byte

	currentToken  *lexer.Token
	previousToken *lexer.Token

	currentScope    []string
	currentGroup    string
	heldId          IdentifierString
	currentFile     string
	currentUniqueId int

	load  *loader.Loader
	lexer *lexer.Lexer

	workspaces map[IdentifierString]*Workspace
}

func NewParser(l *loader.Loader) *Parser {
	p := new(Parser)
	p.load = l

	return p
}

func (p *Parser) Run() error {

	for {
		source, err := p.load.NextWorkspace()
		if err != nil {
			return err
		}
		if source == nil {
			break
		}

		p.currentFile = source.File
		p.code = source.Dsl
		p.lexer = lexer.NewLexer(p.code)

		err = p.lexer.Run()
		if err != nil {
			return err
		}

		newWorks, err := p.runParse()
		if err != nil {
			return err
		}

		for _, w := range newWorks {
			if _, exists := p.workspaces[w.Id()]; exists {
				return fmt.Errorf("duplicate workspace declaration for %s in file %s", w.Id(), p.currentFile)
			}
			p.workspaces[w.Id()] = w
		}
	}

	err := p.Reconcile()
	if err != nil {
		return err
	}

	return nil
}

func (p *Parser) reset() {
	p.currentToken = nil
	p.currentFile = ""
	p.lexer = nil
	p.currentScope = nil
	p.currentToken = &lexer.Token{}
}

func (p *Parser) currentSymbol() string {
	return string(p.currentToken.BytesAt(p.code))
}

func (p *Parser) nextToken() *lexer.Token {
	p.previousToken = p.currentToken
	p.currentToken = p.lexer.NextToken()
	return p.currentToken
}

func (p *Parser) backupToken() {
	if p.previousToken == nil {
		panic("attempt to double backup tokens")
	}
	p.currentToken = p.previousToken
	p.previousToken = nil
	p.lexer.BackupToken()
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
	if p.heldId != "" {
		panic("already holding identifier " + p.heldId + " can't hold " + id)
	}

	p.heldId = id
}

func (p *Parser) claimHeldIdentifier() IdentifierString {
	if p.heldId == "" {
		panic("attempt to claim null identifier")
	}
	id := p.heldId
	p.heldId = ""
	return id
}

func (p *Parser) assignIdentifier(e Entity) {
	if p.heldId != "" {
		e.SetId(p.heldId)
		p.heldId = ""
		return
	}

	typeName := ""
	fallback := ""
	switch obj := e.(type) {
	case *Workspace:
		typeName = "workspace"
		fallback = obj.File
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

func (p *Parser) loadWorkspace(uri string) {
	//p.loader.Load(uri)
}
