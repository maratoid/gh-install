package selector

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/manifoldco/promptui"
	"github.com/maratoid/gh-install/output"
	"github.com/mholt/archiver/v4"
)

type ISelector interface {
	SelectItems() ([]*SelectorItem, error)
}

type Selector struct {
	Kind     string
	Items    []*SelectorItem
	Name     string
	Matcher  string
	Multiple bool
}

func (s *Selector) SelectItems() ([]*SelectorItem, error) {
	var selectedItems []*SelectorItem
	var matches []string

	output.Output().Set(fmt.Sprintf("%s_%s", strings.ReplaceAll(s.Kind, " ", "_"), "matcher"), s.Matcher)
	output.Output().Set(fmt.Sprintf("%s_%s", strings.ReplaceAll(s.Kind, " ", "_"), "multiple"), s.Multiple)

	for _, item := range s.Items {
		if s.Name == "" {
			match, err := regexp.MatchString(s.Matcher, item.Name)
			if err != nil {
				return nil, err
			}
			if match {
				item.Selected = true
				selectedItems = append(selectedItems, item)
				matches = append(matches, item.Name)
			}

		} else {
			if strings.Compare(strings.ToLower(s.Name), strings.ToLower(item.Name)) == 0 {
				item.Selected = true
				selectedItems = append(selectedItems, item)
				matches = append(matches, item.Name)
			}
		}
	}

	if matches == nil {
		matches = make([]string, 0, 1)
	}
	output.Output().Set(fmt.Sprintf("%s_%s", strings.ReplaceAll(s.Kind, " ", "_"), "matches"), matches)

	if !s.Multiple && len(selectedItems) > 1 {
		return nil, fmt.Errorf("more than one item matching '%s' found for %s", s.Matcher, s.Kind)
	}

	if len(selectedItems) == 0 {
		return nil, fmt.Errorf("no %s matching '%s' found", s.Kind, s.Matcher)
	}

	return selectedItems, nil
}

type InteractiveSelector struct {
	Kind         string
	LastSelected int
	Items        []*SelectorItem
	Prompt       string
	Multiple     bool
}

func (s *InteractiveSelector) searcher(input string, index int) bool {
	return fuzzy.Match(input, s.Items[index].Name)
}

func (s *InteractiveSelector) SelectItems() ([]*SelectorItem, error) {
	var templates *promptui.SelectTemplates

	doneItem := MakeSelectorItem(
		"Done",
		false,
		MakeProp("id", -1),
	)

	if s.Multiple {
		if len(s.Items) > 0 && s.Items[0].GetPropInt("id") != doneItem.GetPropInt("id") {
			s.Items = append([]*SelectorItem{doneItem}, s.Items...)
		}

		// Define promptui template
		templates = &promptui.SelectTemplates{
			Help: "Use <Enter> to mark/unmark selection, '↓ ↑ → ←' to navigate. Select 'Done' when finished.",
			Label: `{{ if .Selected }}
						✔
					{{ end }} {{ .Name }}`,
			Active:   "→ {{ if .Selected }}✔ {{ end }}{{ .Name | cyan }}",
			Inactive: "{{ if .Selected }}✔ {{ end }}{{ .Name }}",
		}
	} else {
		// Define promptui template
		templates = &promptui.SelectTemplates{
			Help:     "Use <Enter> to select, '/' to search and '↓ ↑ → ←' to navigate.",
			Active:   "→ {{ .Name | cyan }}",
			Inactive: "{{ .Name }}",
		}
	}
	var prompt promptui.Select

	if s.Multiple {
		prompt = promptui.Select{
			Label:        s.Prompt,
			Items:        s.Items,
			Templates:    templates,
			Size:         10,
			CursorPos:    s.LastSelected,
			HideSelected: true,
		}
	} else {
		prompt = promptui.Select{
			Label:        s.Prompt,
			Items:        s.Items,
			Searcher:     s.searcher,
			Templates:    templates,
			Size:         10,
			CursorPos:    s.LastSelected,
			HideSelected: true,
		}
	}

	selectionIndex, _, err := prompt.Run()
	if err != nil {
		return nil, err
	}

	selectedItem := s.Items[selectionIndex]
	if s.Multiple {
		if selectedItem.GetPropInt("id") != doneItem.GetPropInt("id") {
			selectedItem.Selected = !selectedItem.Selected
			s.LastSelected = selectionIndex
			return s.SelectItems()
		}

		var selectedItems []*SelectorItem
		for _, item := range s.Items {
			if item.Selected {
				selectedItems = append(selectedItems, item)
			}
		}

		return selectedItems, nil
	}

	return []*SelectorItem{selectedItem}, nil
}

func ReleaseSelector(ghClient *api.RESTClient, repo string, version string, interactive bool) (ISelector, error) {
	response := []struct {
		Tag_name string
		Id       int
	}{}
	err := ghClient.Get(fmt.Sprintf("repos/%s/releases", repo), &response)
	if err != nil {
		return nil, err
	}

	var items []*SelectorItem
	for _, val := range response {
		items = append(items, MakeSelectorItem(val.Tag_name, false, MakeProp("id", val.Id)))
	}

	if interactive {
		return &InteractiveSelector{
			Kind:         "release versions",
			Items:        items,
			Prompt:       fmt.Sprintf("Please select %s release tag", repo),
			LastSelected: 0,
			Multiple:     false,
		}, nil
	}

	versionMatcher := version
	if versionMatcher == "latest" {
		response := struct {
			Tag_name string
		}{}

		err := ghClient.Get(fmt.Sprintf("repos/%s/releases/latest", repo), &response)
		if err != nil {
			return nil, err
		}
		versionMatcher = response.Tag_name
	}

	return &Selector{
		Kind:     "release versions",
		Items:    items,
		Matcher:  versionMatcher,
		Multiple: false,
	}, nil
}

func AssetSelector(ghClient *api.RESTClient, repo string,
	releaseId int, name string, matcher string, interactive bool) (ISelector, error) {
	var linkRE = regexp.MustCompile(`<([^>]+)>;\s*rel="([^"]+)"`)
	var items []*SelectorItem
	page := 1
	requestPath := fmt.Sprintf("repos/%s/releases/%d/assets", repo, releaseId)
	findNextPage := func(response *http.Response) (string, bool) {
		for _, m := range linkRE.FindAllStringSubmatch(response.Header.Get("Link"), -1) {
			if len(m) > 2 && m[2] == "next" {
				return m[1], true
			}
		}
		return "", false
	}

	for {
		response, err := ghClient.Request(http.MethodGet, requestPath, nil)
		if err != nil {
			return nil, err
		}
		decoder := json.NewDecoder(response.Body)

		responseData := []struct{ Name string }{}
		err = decoder.Decode(&responseData)
		if err != nil {
			return nil, err
		}
		if err := response.Body.Close(); err != nil {
			return nil, err
		}

		for index, val := range responseData {
			items = append(items, MakeSelectorItem(val.Name, false, MakeProp("id", index)))
		}

		var hasNextPage bool
		if requestPath, hasNextPage = findNextPage(response); !hasNextPage {
			break
		}
		page++
	}

	if interactive {
		return &InteractiveSelector{
			Kind:         "release assets",
			Items:        items,
			Prompt:       fmt.Sprintf("Please select %s asset", repo),
			LastSelected: 0,
			Multiple:     false,
		}, nil
	}

	return &Selector{
		Kind:     "release assets",
		Items:    items,
		Name:     name,
		Matcher:  matcher,
		Multiple: false,
	}, nil
}

func BinarySelector(downloadPath string, name string, matcher string, interactive bool) (ISelector, error) {
	inputStream, err := os.Open(downloadPath)
	if err != nil {
		return nil, err
	}

	var items []*SelectorItem
	_, _, err = archiver.Identify(downloadPath, inputStream)
	if err != nil {
		if err == archiver.ErrNoMatch {
			items = append(items, MakeSelectorItem(
				path.Base(downloadPath),
				false,
				MakeProp("archive", false),
				MakeProp("binType", path.Ext(downloadPath)),
				MakeProp("path", downloadPath),
				MakeProp("id", 0)))

			if interactive {
				return &InteractiveSelector{
					Kind:         "release asset binaries",
					Items:        items,
					Prompt:       "Confirm release binary to be installed",
					LastSelected: 0,
					Multiple:     false,
				}, nil
			}

			return &Selector{
				Kind:     "release asset binaries",
				Items:    items,
				Name:     name,
				Matcher:  path.Base(downloadPath),
				Multiple: false,
			}, nil
		}
		return nil, err
	}

	fileSystem, err := archiver.FileSystem(context.TODO(), downloadPath)
	if err != nil {
		return nil, err
	}

	err = fs.WalkDir(fileSystem, ".", func(fsPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			items = append(items,
				MakeSelectorItem(
					d.Name(),
					false,
					MakeProp("archive", true),
					MakeProp("path", fsPath),
					MakeProp("fs", fileSystem),
					MakeProp("id", 0)))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if interactive {
		return &InteractiveSelector{
			Kind:         "release asset binaries",
			Items:        items,
			Prompt:       "Select binaries to be installed",
			LastSelected: 1,
			Multiple:     true,
		}, nil
	}

	return &Selector{
		Kind:     "release asset binaries",
		Items:    items,
		Name:     name,
		Matcher:  matcher,
		Multiple: false,
	}, nil
}
