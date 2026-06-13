package ecr

func (p *ECRPage) HasFocusedInput() bool {
	return p.searchInput.Focused() || p.createInput.Focused() || p.manualInput.Focused() || p.tagInput.Focused()
}
