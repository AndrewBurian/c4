package parser

import (
	"fmt"

	"go.burian.dev/c4/compiler/lexer"
)

type Workspace struct {
	baseEntity

	Extends string `json:"extends,omitempty"`
	File    string `json:"file,omitempty"`
	Model   *Model `json:"model,omitempty"`
	Views   *Views `json:"views,omitempty"`
}

type Model struct {
	baseEntity

	People          []*Person         `json:"people,omitempty"`
	SoftwareSystems []*SoftwareSystem `json:"software_systems,omitempty"`
}

type Views struct{}

type SoftwareSystem struct {
	baseEntity
}

type Container struct {
	baseEntity
	relationshipEntity
}

type Relationship struct {
	baseEntity
	SourceId      IdentifierString `json:"source_id,omitempty"`
	DestinationId IdentifierString `json:"destination_id,omitempty"`

	ImpliedBasedOn *Relationship `json:"implied_based_on,omitempty"`
}

type Person struct {
	baseEntity
}

type Component struct {
	baseEntity
}

func (p *Parser) runParse() ([]*Workspace, error) {

	var works []*Workspace

	for {

		if p.acceptOne(lexer.TypeEOF) {
			if len(works) < 1 {
				// read at least one workspace
				return nil, p.errExpectedNext().Keywords(KeywordWorkspace)
			}
			return works, nil
		}
		if p.acceptOne(lexer.TypeKeyword) {
			if p.currentKeyword() != KeywordWorkspace {
				return nil, p.errExpectedNext().Keywords(KeywordWorkspace)
			}

			w, err := p.parseWorkspace()
			if err != nil {
				return nil, fmt.Errorf("error parsing workspace:\n> %w", err)
			}
			works = append(works, w)
			continue
		}
		return nil, p.errExpectedNext().Keywords(KeywordWorkspace)

	}
}

func (p *Parser) parseWorkspace() (*Workspace, error) {
	wk := new(Workspace)
	wk.File = p.currentFile
	var err error

	// looking for `extends <path>`
	if p.acceptOne(lexer.TypeKeyword) {

		if key := p.currentKeyword(); key != KeywordExtends {
			return nil, p.errExpectedNext().Keywords(KeywordExtends)
		}

		wk.Extends, err = p.parseString()
		if err != nil {
			return nil, fmt.Errorf("error parsing extension path:\n> %w", err)
		}
	} else {
		// otherwise it's `[name] [description]`
		err = p.parseShortDeclarationSeq(0,
			&wk.Name,
			&wk.Description,
		)
		if err != nil {
			return nil, fmt.Errorf("error parsing workspace declaration:\n> %w", err)
		}
	}

	if p.acceptOne(lexer.TypeTerminator) {
		// weird, but technically valid grammer?
		// just an empty workspace declaration
		return wk, nil
	}

	if !p.acceptOne(lexer.TypeStartBlock) {
		return nil, p.errExpectedNext().Tokens(lexer.TypeStartBlock)
	}

	expectedKeywords := []Keyword{
		KeywordName,
		KeywordDescription,
		KeywordProperties,
		KeywordModel,
		KeywordView,
	}

	for {

		err := p.parseEntityBase(&wk.baseEntity,
			expectedKeywords...,
		)
		if err != nil {
			return nil, fmt.Errorf("error parsing workspace block declaration:\n> %w", err)
		}

		if p.acceptOne(lexer.TypeEndBlock) {
			return wk, nil
		}

		if p.acceptOne(lexer.TypeEOF) {
			return nil, p.errExpectedNext().Tokens(lexer.TypeEndBlock).Keywords(expectedKeywords...)
		}

		if p.acceptOne(lexer.TypeKeyword) {
			switch p.currentKeyword() {
			case KeywordName:
				wk.Name, err = p.parseString()
				if err != nil {
					return nil, p.errExpectedNext().Keywords(expectedKeywords...)
				}
				continue

			case KeywordModel:
				if wk.Model != nil {
					return nil, fmt.Errorf("invalid redefinition of model")
				}
				if wk.Model, err = p.parseModel(); err != nil {
					return nil, fmt.Errorf("error parsing workspace definition:\n> %w", err)
				}
				continue

			case KeywordView:
				if wk.Views != nil {
					return nil, fmt.Errorf("invalid redefinition of views")
				}

				if wk.Views, err = p.parseViews(); err != nil {
					return nil, fmt.Errorf("error parsing views definition:\n> %w", err)
				}
				continue

			default:
				panic("unhandled keyword case " + p.currentKeyword())
			}

		}

	}
}

func (p *Parser) parseModel() (*Model, error) {

	m := new(Model)
	var err error

	if !p.acceptOne(lexer.TypeStartBlock) {
		return nil, p.errExpectedNext().Tokens(lexer.TypeStartBlock)
	}

	for {

		err = p.parseEntityBase(&m.baseEntity,
			KeywordPerson,
			KeywordSoftwareSystem,
			KeywordGroup, // not handled by entity base
		)
		if err != nil {
			return nil, fmt.Errorf("parsing model base definition:\n> %w", err)
		}

		if p.acceptOne(lexer.TypeEndBlock) {
			return m, nil
		}

		expectedKeywords := []Keyword{KeywordGroup, KeywordPerson, KeywordSoftwareSystem}
		if !p.acceptOne(lexer.TypeKeyword) {
			return nil, p.errExpectedNext().Keywords(expectedKeywords...)
		}

		switch p.currentKeyword() {

		case KeywordGroup:
			if err = p.parseModelGroup(m); err != nil {
				return nil, fmt.Errorf("error parsing model:\n> %w", err)
			}

		default:
			return nil, p.errExpectedNext().Keywords(expectedKeywords...)
		}

	}
}

func (p *Parser) parseViews() (*Views, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Parser) parsePerson() (*Person, error) {

	per := new(Person)

	err := p.parseShortDeclarationSeq(1,
		&per.Name,
		&per.Description,
		&per.Tags,
	)
	if err != nil {
		return nil, fmt.Errorf("error parsing person short declaration:\n> %w", err)
	}

	if p.acceptOne(lexer.TypeTerminator) {
		return per, nil
	}

	if !p.acceptOne(lexer.TypeStartBlock) {
		return nil, p.errExpectedNext().Tokens(lexer.TypeStartBlock)
	}

	err = p.parseEntityBase(&per.baseEntity,
		KeywordDescription,
		KeywordTags,
		KeywordUrl,
		KeywordProperties,
		KeywordPerspectives,
	)
	if err != nil {
		return nil, fmt.Errorf("error parsing person block declaration:\n> %w", err)
	}

	if !p.acceptOne(lexer.TypeEndBlock) {
		p.errExpectedNext().Tokens(lexer.TypeEndBlock)
	}

	return per, nil

}

func (p *Parser) parseSoftwareSys() (*SoftwareSystem, error) {

	ss := new(SoftwareSystem)

	err := p.parseShortDeclarationSeq(1,
		&ss.Name,
		&ss.Description,
		&ss.Tags,
	)
	if err != nil {
		return nil, fmt.Errorf("error parsing softwaresystem short delcaration:\n> %w", err)
	}

	if p.acceptOne(lexer.TypeTerminator) {
		return ss, nil
	}

	if !p.acceptOne(lexer.TypeStartBlock) {
		return nil, p.errExpectedNext().Tokens(lexer.TypeStartBlock)
	}

	// we have to loop this one since we have a keyword that base won't
	// handle for us, and it might come up at any time
	for {

		err = p.parseEntityBase(&ss.baseEntity,
			KeywordContainer,
			KeywordDescription,
			KeywordTags,
			KeywordUrl,
			KeywordProperties,
			KeywordPerspectives,
			KeywordGroup, // unhandled by pase parser
		)
		if err != nil {
			return nil, fmt.Errorf("error parsing softwaresystem body:\n> %w", err)
		}

		if p.acceptOne(lexer.TypeEndBlock) {
			return ss, nil
		}

		if p.acceptOne(lexer.TypeKeyword) {
			if p.currentKeyword() != KeywordGroup {
				panic("unhandled keyword by entity base parser should have errored")
			}

			return nil, fmt.Errorf("inimplemented: software system group")
		}
	}
}

// Handles either the remainder of a relationship declaration
// or up to the assignment operator of an entity assignment
func (p *Parser) parseEntityNameOrRelationship(entity Relatable) (err error) {
	// pick up entity name assignments and relationships
	// foo = (keyword)
	// bar -> (identifier)
	if !p.acceptOne(lexer.TypeRelationship) {
		if p.acceptOne(lexer.TypeAssignment) {
			return nil
		}
		return p.errExpectedNext().Tokens(lexer.TypeRelationship, lexer.TypeAssignment)
	}

	r, err := p.parseRelationship(p.claimHeldIdentifier())
	if err != nil {
		return fmt.Errorf("error parsing model:\n> %w", err)
	}
	entity.SetRelationship(r)

	return nil
}

func (p *Parser) parseModelGroup(m *Model) error {
	if !p.acceptOne(lexer.TypeString) {
		return p.errExpectedNext().Tokens(lexer.TypeString)
	}

	currentGroup := p.currentSymbol()

	if !p.acceptOne(lexer.TypeStartBlock) {
		return p.errExpectedNext().Tokens(lexer.TypeStartBlock)
	}

	for {

		if p.acceptOne(lexer.TypeEndBlock) {
			return nil
		}

		if p.acceptIdentifierString() {
			if !p.acceptOne(lexer.TypeAssignment) {
				return p.errExpectedNext().Tokens(lexer.TypeAssignment)
			}
		}

		if !p.acceptOne(lexer.TypeKeyword) {
			return p.errExpectedNext().Keywords(KeywordPerson, KeywordSoftwareSystem)
		}

		switch p.currentKeyword() {

		case KeywordPerson:
			pers, err := p.parsePerson()
			if err != nil {
				return fmt.Errorf("error parsing model group %s:\n> %w", currentGroup, err)
			}
			pers.Group = currentGroup
			p.assignIdentifier(pers)
			m.People = append(m.People, pers)
			m.Add(pers)

		case KeywordSoftwareSystem:
			ss, err := p.parseSoftwareSys()
			if err != nil {
				return fmt.Errorf("error parsing model group %s:\n> %w", currentGroup, err)
			}
			ss.Group = currentGroup
			p.assignIdentifier(ss)
			m.Add(ss)
			m.SoftwareSystems = append(m.SoftwareSystems, ss)

		default:
			return p.errExpectedNext().Keywords(KeywordPerson, KeywordSoftwareSystem)
		}
	}
}

func (p *Parser) parseSoftwareSysGroup(ss *SoftwareSystem) error {
	if !p.acceptOne(lexer.TypeString) {
		return p.errExpectedNext().Tokens(lexer.TypeString)
	}

	p.enterGroup(p.currentSymbol())
	defer p.leaveGroup()

	if !p.acceptOne(lexer.TypeStartBlock) {
		return p.errExpectedNext().Tokens(lexer.TypeStartBlock)
	}

	for {

		if p.acceptOne(lexer.TypeEndBlock) {
			return nil
		}

		if !p.acceptOne(lexer.TypeKeyword) {
			return p.errExpectedNext().Keywords(KeywordPerson, KeywordSoftwareSystem)
		}

		if p.currentKeyword() != KeywordContainer {
			return p.errExpectedNext().Keywords(KeywordContainer)
		}

		c, err := p.parseContainer()
		if err != nil {
			return fmt.Errorf("error parsing software system group:\n> %w", err)
		}
		p.assignGroup(c)
		ss.Add(c)
	}
}

func (p *Parser) parseContainer() (*Container, error) {
	c := new(Container)

	err := p.parseShortDeclarationSeq(1,
		&c.Name,
		&c.Description,
		&c.Technology,
		&c.Tags,
	)
	if err != nil {
		return nil, fmt.Errorf("error parsing container short declaration:\n> %w", err)
	}

	if p.acceptOne(lexer.TypeTerminator) {
		return c, nil
	}

	if !p.acceptOne(lexer.TypeStartBlock) {
		return nil, p.errExpectedNext().Tokens(lexer.TypeStartBlock)
	}

	allowedContainerProps := []Keyword{
		KeywordComponent,
		KeywordDescription,
		KeywordTechnology,
		KeywordTags,
		KeywordUrl,
		KeywordProperties,
		KeywordPerspectives,
	}

	if err := p.parseEntityBase(&c.baseEntity, allowedContainerProps...); err != nil {
		return nil, fmt.Errorf("error parsing container:\n> %w", err)
	}

	if !p.acceptOne(lexer.TypeEndBlock) {
		return nil, p.errExpectedNext().Tokens(lexer.TypeEndBlock)
	}

	return c, nil

}

func (p *Parser) parseComponent() (*Component, error) {
	c := new(Component)

	err := p.parseShortDeclarationSeq(1,
		&c.Name,
		&c.Description,
		&c.Technology,
		&c.Tags,
	)
	if err != nil {
		return nil, fmt.Errorf("error parsing component short declaration:\n> %w", err)
	}

	if p.acceptOne(lexer.TypeTerminator) {
		return c, nil
	}

	if !p.acceptOne(lexer.TypeStartBlock) {
		return nil, p.errExpectedNext().Tokens(lexer.TypeStartBlock)
	}

	allowedComponentProps := []Keyword{
		KeywordDescription,
		KeywordTechnology,
		KeywordTags,
		KeywordUrl,
		KeywordProperties,
		KeywordPerspectives,
	}

	if err := p.parseEntityBase(&c.baseEntity, allowedComponentProps...); err != nil {
		return nil, fmt.Errorf("error parsing container:\n> %w", err)
	}

	if !p.acceptOne(lexer.TypeEndBlock) {
		return nil, p.errExpectedNext().Tokens(lexer.TypeEndBlock)
	}

	return c, nil
}

func (p *Parser) parseRelationship(from IdentifierString) (*Relationship, error) {
	r := new(Relationship)
	if !p.acceptIdentifierString() {
		return nil, p.errExpectedNext().Tokens(lexer.TypeIdentifier)
	}
	r.SourceId = from
	r.DestinationId = p.claimHeldIdentifier()

	err := p.parseShortDeclarationSeq(0,
		&r.Description,
		&r.Technology,
		&r.Tags,
	)
	if err != nil {
		return nil, fmt.Errorf("error parsing relationship short declaration:\n> %w", err)
	}

	if p.acceptOne(lexer.TypeTerminator) {
		return r, nil
	}

	if !p.acceptOne(lexer.TypeStartBlock) {
		return nil, p.errExpectedNext().Tokens(lexer.TypeStartBlock)
	}

	err = p.parseEntityBase(&r.baseEntity,
		KeywordTags,
		KeywordUrl,
		KeywordProperties,
		KeywordPerspectives,
		KeywordTechnology,
	)
	if err != nil {
		return nil, fmt.Errorf("error parsing relationship body:\n> %w", err)
	}

	if !p.acceptOne(lexer.TypeEndBlock) {
		return nil, p.errExpectedNext().Tokens(lexer.TypeEndBlock)
	}

	return r, nil
}
