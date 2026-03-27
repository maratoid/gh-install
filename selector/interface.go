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

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/mholt/archiver/v4"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type SelectorKind int

const (
	Release SelectorKind = iota
	Asset
	Binary
)

func (ss SelectorKind) String() string {
	switch ss {
	case Release:
		return "release_selector"
	case Asset:
		return "asset_selector"
	case Binary:
		return "binary_selector"
	default:
		return fmt.Sprintf("Unknown(%d)", ss)
	}
}

type ISelector interface {
	GetKind() SelectorKind
	Run() ([]*SelectorItem, error)
}

func ReleaseSelector(ghClient *api.RESTClient, repo string, version string, interactive bool) (ISelector, error) {
	log.Info().
		Str("repository", repo).
		Msg("getting Github repository releases")

	response := []struct {
		Tag_name string
		Id       int
	}{}
	err := ghClient.Get(fmt.Sprintf("repos/%s/releases", repo), &response)
	if err != nil {
		return nil, err
	}

	items := make(map[string]*SelectorItem)
	itemsOrder := make([]string, 0, len(response))
	for _, val := range response {
		log.Debug().
			Str("repository", repo).
			Str("release tag", val.Tag_name).
			Int("release id", val.Id).
			Msg("got release...")
		itemsOrder = append(itemsOrder, val.Tag_name)
		items[val.Tag_name] = MakeSelectorItem(val.Tag_name, false).SetId(val.Id)
	}

	if interactive {
		return &InteractiveSelector{
			Kind:      Release,
			Items:     items,
			ItemOrder: itemsOrder,
			Prompt:    fmt.Sprintf("Please select %s release tag", repo),
			Single:    true,
		}, nil
	}

	versionMatcher := version
	if versionMatcher == "latest" {
		log.Debug().
			Str("repository", repo).
			Msg("needed release version is 'latest', getting actual version")
		response := struct {
			Tag_name string
		}{}

		err := ghClient.Get(fmt.Sprintf("repos/%s/releases/latest", repo), &response)
		if err != nil {
			return nil, err
		}
		versionMatcher = response.Tag_name
		log.Debug().
			Str("repository", repo).
			Str("release tag", versionMatcher).
			Msg("got 'latest' release")
	}

	log.Info().
		Str("repository", repo).
		Msgf("using release %s", versionMatcher)

	return &Selector{
		Kind:          Release,
		Items:         items,
		RegexpMatcher: versionMatcher,
		Single:        true,
	}, nil
}

func AssetSelector(ghClient *api.RESTClient, repo string,
	releaseId int, name string, matcher string, interactive bool) (ISelector, error) {
	var linkRE = regexp.MustCompile(`<([^>]+)>;\s*rel="([^"]+)"`)
	itemsOrder := make([]string, 0, 10)
	items := make(map[string]*SelectorItem)
	page := 1
	requestPath := fmt.Sprintf("repos/%s/releases/%d/assets", repo, releaseId)

	log.Info().
		Str("repository", repo).
		Int("release id", releaseId).
		Str("asset matching name", name).
		Str("asset matching regexp", matcher).
		Msg("getting release assets")

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
			itemsOrder = append(itemsOrder, val.Name)
			items[val.Name] = MakeSelectorItem(val.Name, false).SetId(index)
			log.Debug().
				Str("repository", repo).
				Int("release id", releaseId).
				Str("asset name", val.Name).
				Int("asset index (id)", index).
				Msg("got release asset")
		}

		var hasNextPage bool
		if requestPath, hasNextPage = findNextPage(response); !hasNextPage {
			log.Debug().
				Str("repository", repo).
				Int("release id", releaseId).
				Msg("end of asset list")
			break
		}
		log.Debug().
			Str("repository", repo).
			Int("release id", releaseId).
			Msg("getting next page of release assets")
		page++
	}

	if interactive {
		return &InteractiveSelector{
			Kind:      Asset,
			Items:     items,
			ItemOrder: itemsOrder,
			Prompt:    fmt.Sprintf("Please select %s asset", repo),
			Single:    true,
		}, nil
	}

	return &Selector{
		Kind:          Asset,
		Items:         items,
		NamesMatcher:  []string{name},
		RegexpMatcher: matcher,
		Single:        true,
	}, nil
}

func BinarySelector(downloadPath string, names []string, matcher string, interactive bool) (ISelector, error) {
	log.Info().
		Str("asset download path", downloadPath).
		Array("asset matching binary names", func() *zerolog.Array {
			arr := zerolog.Arr()
			for _, i := range names {
				arr = arr.Str(i)
			}
			return arr
		}()).
		Str("asset matching binary regexp", matcher).
		Msg("getting release asset binaries")

	inputStream, err := os.Open(downloadPath)
	if err != nil {
		return nil, err
	}

	itemsOrder := make([]string, 0, 10)
	items := make(map[string]*SelectorItem)
	_, _, err = archiver.Identify(downloadPath, inputStream)
	if err != nil {
		if err == archiver.ErrNoMatch {
			itemsOrder = append(itemsOrder, path.Base(downloadPath))
			items[path.Base(downloadPath)] = MakeSelectorItem(
				path.Base(downloadPath),
				false).
				SetCompressed(false).
				SetBinaryType(BinaryTypeFromPath(downloadPath)).
				SetDownloadPath(downloadPath).
				SetId(0)

			if interactive {
				return &InteractiveSelector{
					Kind:      Binary,
					Items:     items,
					ItemOrder: itemsOrder,
					Prompt:    "Confirm release binary to be installed",
					Single:    true,
				}, nil
			}

			return &Selector{
				Kind:          Binary,
				Items:         items,
				NamesMatcher:  names,
				RegexpMatcher: path.Base(downloadPath),
				Single:        true,
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
			itemsOrder = append(itemsOrder, d.Name())
			items[d.Name()] = MakeSelectorItem(d.Name(), false).
				SetId(0).
				SetCompressed(true).
				SetFsPath(fsPath).
				SetFs(fileSystem)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if interactive {
		return &InteractiveSelector{
			Kind:      Binary,
			Items:     items,
			ItemOrder: itemsOrder,
			Prompt:    "Select binaries to be installed",
			Single:    false,
		}, nil
	}

	return &Selector{
		Kind:          Binary,
		Items:         items,
		NamesMatcher:  names,
		RegexpMatcher: matcher,
		Single:        false,
	}, nil
}
