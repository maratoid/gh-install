package selector

import (
	"atomicgo.dev/keyboard/keys"
	"github.com/pterm/pterm"
)

type InteractiveSelector struct {
	Kind      SelectorKind
	ItemOrder []string
	Items     map[string]*SelectorItem
	Prompt    string
	Single    bool
}

func (s *InteractiveSelector) showPrompt() ([]string, error) {
	if s.Single {
		selectedItem, err := pterm.DefaultInteractiveSelect.
			WithOptions(s.ItemOrder).
			WithDefaultText(s.Prompt).Show()
		if err != nil {
			return nil, err
		}
		return []string{selectedItem}, nil
	}

	selectedItems, err := pterm.DefaultInteractiveMultiselect.
		WithOptions(s.ItemOrder).
		WithDefaultText(s.Prompt).
		WithKeyConfirm(keys.Enter).
		WithKeySelect(keys.Space).
		WithFilter(false).
		Show()
	if err != nil {
		return nil, err
	}
	return selectedItems, nil
}

func (s *InteractiveSelector) Run() ([]*SelectorItem, error) {
	selectedNames, err := s.showPrompt()
	if err != nil {
		return nil, err
	}
	if s.Single {
		return []*SelectorItem{s.Items[selectedNames[0]]}, nil
	}

	var selectedItems []*SelectorItem
	for _, selectedName := range selectedNames {
		selectedItems = append(selectedItems, s.Items[selectedName])
	}

	return selectedItems, nil
}

func (s *InteractiveSelector) GetKind() SelectorKind {
	return s.Kind
}
