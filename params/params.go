package params

import "github.com/alecthomas/kong"

type CLI struct {
	Repository              string            `arg:"" help:"Github repository in OWNER/REPOSITORY_NAME format."`
	Interactive             bool              `default:"false" short:"i" help:"Use interactive installation. If true, all non-log related flags are ignored." group:"Interactive Mode"`
	ReleaseVersion          string            `default:"latest" short:"v" help:"Repository release tag (version) to install." group:"Non-interactive Mode"`
	ReleaseAsset            string            `optional:"" short:"a" help:"Name of repository release asset to download. If not set, --release-asset-regexp is used." group:"Non-interactive Mode"`
	ReleaseAssetRegexp      string            `default:"${release_asset_regexp}" short:"A" help:"Regular expression matching release asset to download." group:"Non-interactive Mode"`
	AssetBinaries           []string          `optional:"" short:"b" help:"If release asset is an archive - names of a binaries in the archive to install. If not set, --install-binary-regexp is used." group:"Non-interactive Mode"`
	AssetBinariesRegexp     string            `optional:"" short:"B" help:"If release asset is an archive - regular expression matching binaries in the archive to install. If not set, repository name is used." group:"Non-interactive Mode"`
	TargetPath              string            `default:"${install_path}" short:"p" type:"path" help:"Target installation directory."`
	TargetBinaries          map[string]string `optional:"" short:"t" help:"Rename binaries installed at target path, \"<asset archive binary | asset>=<renamed binary>;...\"" group:"Non-interactive Mode"`
	TargetPathCreate        bool              `default:"true" negatable:"" help:"Create target installation directory if it does not exist." group:"Non-interactive Mode"`
	TargetBinariesOverwrite bool              `default:"false" short:"o" help:"Overwrite target binaries." group:"Non-interactive Mode"`
	LogLevel                string            `default:"info" enum:"error,warn,info,debug" short:"l" help:"Log level."`
	LogFormat               string            `default:"console" enum:"console,json" short:"f" help:"Log output format."`
	LogQuietInteractive     bool              `default:"true" negatable:"" help:"Quiet log in interactive mode" group:"Interactive Mode"`
	Version                 kong.VersionFlag  `help:"Show version." env:""`
}
