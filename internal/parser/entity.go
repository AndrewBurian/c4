package parser

import (
	"fmt"
	"strings"

	"go.burian.dev/c4arch/internal/lexer"
)

type Entity interface {
	Id() IdentifierString
	SetId(IdentifierString)

	SetGroup(string)
}

type baseEntity struct {
	LocalId          IdentifierString
	FullyQualifiedId IdentifierString

	Parent Entity

	Name         string
	Description  string
	Group        string
	Properties   map[string]string
	Perspectives map[string]string
	Tags         []string
	Technology   string
	Url          string

	relationshipEntity
	childEntities
}

type ParentEntity interface {
	Add(Entity) error
}

type Relatable interface {
	SetRelationship(*Relationship)
}

type relationshipEntity struct {
	Relationships []*Relationship
}

func (re *relationshipEntity) SetRelationship(r *Relationship) {
	re.Relationships = append(re.Relationships, r)
}

func (b *baseEntity) Id() IdentifierString {
	if b.FullyQualifiedId != "" {
		return b.FullyQualifiedId
	}
	return b.LocalId
}
func (b *baseEntity) SetId(id IdentifierString) {
	b.LocalId = id
}
func (b *baseEntity) SetGroup(name string) {
	b.Group = name
}

type childEntities struct {
	NamedEntities map[IdentifierString]Entity
}

func (ch *childEntities) Add(e Entity) error {
	if _, exists := ch.NamedEntities[e.Id()]; exists {
		return fmt.Errorf("redefining identifier %s", e.Id())
	}
	if ch.NamedEntities == nil {
		ch.NamedEntities = make(map[IdentifierString]Entity)
	}
	ch.NamedEntities[e.Id()] = e
	return nil
}

func (p *Parser) parseShortDeclarationSeq(must int, targets ...any) error {
	finished := 0
parsing:
	for ti := range targets {

		switch target := targets[ti].(type) {

		case *string:
			if !p.acceptOne(lexer.TypeString) {
				return nil
			}
			p.backupToken()
			str, err := p.parseString()
			if err != nil {
				break parsing
			}
			*target = str

		case *[]string:
			if !p.acceptOne(lexer.TypeString) {
				break parsing
			}
			tags, err := p.parseTags()
			if err != nil {
				return err
			}
			*target = append(*target, tags...)

		case *IdentifierString:
			if !p.acceptIdentifierString() {
				break parsing
			}
			*target = p.claimHeldIdentifier()

		default:
			panic("unknown short declaration type")
		}
		finished++
	}

	if finished < must {
		return fmt.Errorf("did not parse enough arguments in short declaration")
	}
	return nil
}

func (p *Parser) parseEntityBase(e *baseEntity, allowed ...Keyword) error {

	allowedKeyword := func(check Keyword, againt []Keyword) bool {
		for i := range againt {
			if check == againt[i] {
				return true
			}
		}
		return false
	}

	for {
		if p.acceptOne(lexer.TypeIdentifier) {
			if err := p.parseEntityNameOrRelationship(e); err != nil {
				return fmt.Errorf("error parsing entity: %w", err)
			}
			continue
		}

		if p.acceptOne(lexer.TypeRelationship) {
			rel, err := p.parseRelationship("this")
			if err != nil {
				return fmt.Errorf("error parsing entity: %w", err)
			}
			e.SetRelationship(rel)
			continue
		}

		if p.acceptOne(lexer.TypeKeyword) {
			if !allowedKeyword(p.currentKeyword(), allowed) {
				return p.errExpectedNext().Keywords(allowed...).
					Tokens(lexer.TypeIdentifier)
			}

			switch p.currentKeyword() {

			case KeywordDescription:
				if e.Description != "" {
					return fmt.Errorf("illegal redeclaration of description in body")
				}
				desc, err := p.parseOneOrMoreLineString()
				if err != nil {
					return err
				}
				e.Description = desc

				if !p.acceptOne(lexer.TypeTerminator) {
					return p.errExpectedNext().Tokens(lexer.TypeTerminator)
				}
				continue

			case KeywordTags:
				if len(e.Tags) > 0 {
					return fmt.Errorf("illegal redeclaratipn of tags in body")
				}
				newTags, err := p.parseTags()
				if err != nil {
					return fmt.Errorf("error parsing entity: %w", err)
				}
				e.Tags = append(e.Tags, newTags...)
				continue

			case KeywordProperties:
				if len(e.Properties) > 0 {
					return fmt.Errorf("illegal dupluicate declaration of properties")
				}
				props, err := p.parseProperties()
				if err != nil {
					return fmt.Errorf("error parsing entity: %w", err)
				}
				e.Properties = props
				continue

			case KeywordPerspectives:
				if len(e.Perspectives) > 0 {
					return fmt.Errorf("illegal dupluicate declaration of perspectives")
				}
				props, err := p.parseProperties()
				if err != nil {
					return fmt.Errorf("error parsing entity: %w", err)
				}
				e.Properties = props
				continue

			case KeywordThis:
				r, err := p.parseRelationship("this")
				if err != nil {
					return fmt.Errorf("error parsing entity: %w", err)
				}
				e.SetRelationship(r)
				continue

			case KeywordPerson:
				pers, err := p.parsePerson()
				if err != nil {
					return fmt.Errorf("error parsing entity: %w", err)
				}
				p.assignIdentifier(pers)
				p.assignGroup(pers)
				e.Add(pers)
				continue

			case KeywordSoftwareSystem:
				ss, err := p.parseSoftwareSys()
				if err != nil {
					return fmt.Errorf("error parsing entity: %w", err)
				}
				p.assignIdentifier(ss)
				p.assignGroup(ss)
				e.Add(ss)
				continue

			case KeywordContainer:
				cont, err := p.parseContainer()
				if err != nil {
					return fmt.Errorf("error parsing entity: %w", err)
				}
				p.assignIdentifier(cont)
				p.assignGroup(cont)
				e.Add(cont)
				continue

			case KeywordComponent:
				comp, err := p.parseComponent()
				if err != nil {
					return fmt.Errorf("error parsing entity: %w", err)
				}
				p.assignIdentifier(comp)
				p.assignGroup(comp)
				e.Add(comp)
				continue

			case KeywordTechnology:
				err := p.parseSimpleValue("technology", &e.Technology)
				if err != nil {
					return err
				}
				continue

			case KeywordUrl:
				err := p.parseSimpleValue("url", &e.Url)
				if err != nil {
					return err
				}
				continue

			case KeywordName:
				err := p.parseSimpleValue("name", &e.Name)
				if err != nil {
					return err
				}
				continue

			default:
				// default is an interesting case here
				// it's a keyword that's allowed, but we don't
				// know how to handle
				// kick it back to the caller with no error

				// backup because we consumed the keyword getting
				// into this switch
				p.backupToken()
				return nil
			}

		}

		// not a token we know how to deal with
		// also return to caller\
		return nil

	}
}

// Simple values are single strings following keywords, and ending in terminators
// e.g. name 'foo';
// returns a helpful error message if you're trying to double-write
func (p *Parser) parseSimpleValue(name string, str *string) error {
	if *str != "" {
		return fmt.Errorf("illegal redeclaration of %s in block", name)
	}
	val, err := p.parseString()
	if err != nil {
		return fmt.Errorf("error parsing %s in block: %w", name, err)
	}
	*str = val
	if !p.acceptOne(lexer.TypeTerminator) {
		return p.errExpectedNext().Tokens(lexer.TypeTerminator)
	}
	return nil
}

func (p *Parser) parseProperties() (map[string]string, error) {
	props := make(map[string]string)
	for p.acceptOne(lexer.TypeString) {
		if p.acceptOne(lexer.TypeEndBlock) {
			break
		}

		key, err := p.parseString()
		if err != nil {
			return nil, fmt.Errorf("error parsing key for key-value pairs: %w", err)
		}

		if !p.acceptOne(lexer.TypeString) {
			return nil, p.errExpectedNext().Tokens(lexer.TypeString)
		}
		props[key], err = p.parseOneOrMoreLineString()
		if err != nil {
			return nil, fmt.Errorf("error parsing property value: %w", err)
		}

		if !p.acceptOne(lexer.TypeTerminator) {
			return nil, p.errExpectedNext().Tokens(lexer.TypeTerminator)
		}
	}

	return props, nil
}

func (p *Parser) parseTags() ([]string, error) {
	// this is either a single string with commas in it, or multiple strings
	tags := make([]string, 0, 1)
	for p.acceptOne(lexer.TypeString) {
		tagStr := p.currentSymbol()

		if strings.Contains(tagStr, ",") {
			// TODO handle escaped comma
			if len(tags) > 0 {
				return nil, fmt.Errorf("mixed comma separated and space separated tags are not allowed")
			}

			dirtyTags := strings.Split(tagStr, ",")
			for i := range dirtyTags {
				dirtyTags[i] = p.cleanString(dirtyTags[i])
			}
			tags = append(tags, dirtyTags...)
		}
		tags = append(tags, p.cleanString(tagStr))
	}
	if !p.acceptOne(lexer.TypeTerminator) {
		p.errExpectedNext().Tokens(lexer.TypeTerminator)
	}

	return tags, nil
}
