package checker

type Checker struct {
}

func (p *Checker) CheckWorkspaces([]*Workspace) error {
	for _, w := range p.workspaces {
		p.reconcileWorkspace(w)
	}

	return nil
}

func (p *Parser) reconcileWorkspace(w *Workspace) {
	p.reconcileModel(w.Model)
}

func (p *Parser) reconcileModel(m *Model) {

	// make sure all objects have full identifiers

	// for _, r := range m.relationships {

	// }
}
