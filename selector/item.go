package selector

import (
	"io/fs"
)


// selection item with collection of IProperty properties
type SelectorItem struct {
	Name string
	Selected bool
	Properties map[string]interface{}
}

func (i SelectorItem) GetProp(key string) interface{} {
	return i.Properties[key]
}
func (i SelectorItem) GetPropStr(key string) string {
	return i.Properties[key].(string)
}
func (i SelectorItem) GetPropInt(key string) int {
	return i.Properties[key].(int)
}
func (i SelectorItem) GetPropBool(key string) bool {
	return i.Properties[key].(bool)
}
func (i SelectorItem) GetPropFs(key string) fs.FS {
	return i.Properties[key].(fs.FS)
}
func (i SelectorItem) SetProp(key string, value interface{}) {
    i.Properties[key] = value
}

type PropertyPair struct {
	Key string
	Value interface{}
}
func MakeProp(key string, val interface{}) PropertyPair {
	return PropertyPair{
		Key: key,
		Value: val,
	}
}

func MakeSelectorItem(name string, selected bool, properties ...PropertyPair) *SelectorItem {
	propMap := make(map[string]interface{})
	for _, property := range properties {
		propMap[property.Key] = property.Value
	}

	return &SelectorItem{
		Name: name,
		Selected: selected,
		Properties: propMap,
	}
}
