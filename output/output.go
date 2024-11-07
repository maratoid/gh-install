package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	hpretty "github.com/gobs/pretty"
	"github.com/tidwall/pretty"
	"golang.org/x/term"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var lock = &sync.Mutex{}

type OutputMap struct {
	sync.RWMutex
	content map[string]interface{}
}

var outputMap *OutputMap

func Output() *OutputMap {
	if outputMap == nil {
		lock.Lock()
		defer lock.Unlock()
		if outputMap == nil {
			outputMap = &OutputMap{
				content: make(map[string]interface{}),
			}
		}
	}

	return outputMap
}

func (c *OutputMap) Set(key string, value any) {
	c.Lock()
	c.content[key] = value
	c.Unlock()
}

func (c *OutputMap) Get(key string) any {
	c.RLock()
	value, _ := c.content[key]
	c.RUnlock()
	return value
}

func (c *OutputMap) Print(asJson bool) {
	c.RLock()
	defer c.RUnlock()

	if asJson {
		var jsonOut []byte

		out, err := json.Marshal(&c.content)
		if err != nil {
			out = []byte(fmt.Sprintf("{\"error\": \"%s\"}", err))
		}

		if term.IsTerminal(int(os.Stdout.Fd())) {
			jsonOut = pretty.Color(pretty.Pretty(out), nil)
		} else {
			jsonOut = pretty.Pretty(out)
		}

		fmt.Fprintln(os.Stdout, string(jsonOut))
	} else {
		printer := hpretty.NewTabPrinter(2)
		printer.TabWidth(40)
		for key, value := range c.content {
			printer.Print(cases.Title(language.Und).String(strings.ReplaceAll(key, "_", " ")))

			var printVal string
			if s, ok := value.(map[string]string); ok {
				for sKey, sValue := range s {
					printVal = fmt.Sprintf("%s%v (%v), ", printVal, sKey, sValue)
				}
				printVal = strings.TrimSuffix(printVal, ", ")
			} else {
				printVal = fmt.Sprintf("%v", value)
			}
			printer.Print(printVal)
		}

	}
}
