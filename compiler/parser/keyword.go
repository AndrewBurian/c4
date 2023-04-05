package parser

import (
	"strings"

	"go.burian.dev/c4/compiler/lexer"
)

type Keyword string

const (
	KeywordWorkspace = Keyword("workspace")
	KeywordExtends   = Keyword("extends")
	KeywordModel     = Keyword("model")
	KeywordView      = Keyword("view")

	KeywordPerson         = Keyword("person")
	KeywordSoftwareSystem = Keyword("softwaresystem")
	KeywordContainer      = Keyword("container")
	KeywordComponent      = Keyword("component")
	KeywordGroup          = Keyword("group")

	KeywordPerspectives = Keyword("perspectives")
	KeywordTags         = Keyword("tags")
	KeywordDescription  = Keyword("description")
	KeywordName         = Keyword("name")
	KeywordProperties   = Keyword("properties")
	KeywordTechnology   = Keyword("technology")
	KeywordUrl          = Keyword("url")
	KeywordThis         = Keyword("this")

	KeywordStyle = Keyword("style")
)

func (p *Parser) currentKeyword() Keyword {
	if !p.currentToken.Is(lexer.TypeKeyword) {
		panic("fetch non-keyword")
	}

	key := Keyword(strings.ToLower(string(p.currentSymbol())))
	return Keyword(key)
	var valid bool

	switch key {
	case KeywordWorkspace:
		fallthrough
	case KeywordModel:
		fallthrough
	case KeywordSoftwareSystem:
		fallthrough
	case KeywordContainer:
		fallthrough
	case KeywordComponent:
		fallthrough
	case KeywordGroup:
		fallthrough
	case KeywordPerspectives:
		fallthrough
	case KeywordTags:
		fallthrough
	case KeywordDescription:
		fallthrough
	case KeywordThis:
		fallthrough
	case KeywordTechnology:
		fallthrough
	case KeywordUrl:
		valid = true
	}

	if !valid {
		panic("invalid keyword: " + key)
	}
	return Keyword(key)
}
