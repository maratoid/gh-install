package main

import (
	"os"

	"github.com/alecthomas/kong"
	"github.com/maratoid/gh-install/cmd"
)

func main() {

	var cli cmd.RootCLI

	ctx := kong.Parse(&cli,
		kong.Name("gh-install"),
		kong.Description(`Install binaries for a Github repository release interactively or non-interactively.  
			Intended for quickly installing release binaries for projects that do not distribute 
			using Homebrew or other package managers.`),
		kong.DefaultEnvars(cmd.GetEnvPrefix()),
		kong.PostBuild(cmd.PostBuild),
		kong.Vars{
			"release_asset_regexp": cmd.GetDefaultAssetRegexp(),
			"install_path":         cmd.GetDefaultTargetPath(),
			"version":              "2.0.0",
		})

	err := ctx.Run()
	if err != nil {
		os.Exit(1)
	}
}
