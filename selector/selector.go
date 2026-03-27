package selector

import (
	"fmt"
	"regexp"
	"strings"
)

type Selector struct {
	Kind          SelectorKind
	Items         map[string]*SelectorItem
	NamesMatcher  []string
	RegexpMatcher string
	Single        bool
}

func (s *Selector) Run() ([]*SelectorItem, error) {
	var selectedItems []*SelectorItem

	for itemName, item := range s.Items {
		if len(s.NamesMatcher) == 0 {
			match, err := regexp.MatchString(s.RegexpMatcher, itemName)
			if err != nil {
				return nil, err
			}
			if match {
				item.Selected = true
				selectedItems = append(selectedItems, item)
			}
		} else {
			for _, name := range s.NamesMatcher {
				if strings.Compare(strings.ToLower(name), strings.ToLower(itemName)) == 0 {
					item.Selected = true
					selectedItems = append(selectedItems, item)
				}
			}
		}
	}

	if len(selectedItems) == 0 {
		return nil, fmt.Errorf("no %s matches for names '%s' or regexp '%s' found", s.Kind.String(), s.NamesMatcher, s.RegexpMatcher)
	}
	return selectedItems, nil
}

func (s *Selector) GetKind() SelectorKind {
	return s.Kind
}
