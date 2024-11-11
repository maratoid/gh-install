# gh-install
Github client extension for [installing Github repository releases](https://maratg.com/posts/gh-install-extension/)

## Installation

```
 gh extension install maratoid/gh-install
```

## Help

```
$ gh install --help
Install binaries for a Github repository release interactively or non-interactively.
			Intended for quickly installing release binaries for projects that do not distribute
			using Homebrew or other package managers.

Usage:
  gh install owner/repository [flags]

Flags:
  -b, --binary string           install release asset archive binary name. If empty, '--binary-regex' is used.
      --binary-regex string     lookup regexp for release asset archive binary. If empty, repository name is used.
  -d, --download string         name for release asset to download. If empty, '--download-regex' is used.
      --download-regex string   lookup regexp for release asset to download. (default "^.*(?:arm64.+darwin|darwin.+arm64)+.*$")
  -h, --help                    help for gh
  -i, --interactive             Use interactive installation. If true, all other flags are ignored
  -j, --json                    JSON output
      --no-create               Do not create target installation directory if it does not exist.
  -p, --path string             Target installation directory. (default "/Users/maratoid/.local/bin")
  -t, --tag string              release tag (version) to install. (default "latest")
  -v, --version                 version for gh
```

Alternatively, set parameter values via environment variables with `GH_INSTALL_` prefix. E.g.:

* `GH_INSTALL_PATH=<some path> gh install ...` is equivalent to `gh install --path <some path>`
* `GH_INSTALL_DOWNLOAD_REGEX=<some regex> GH_INSTALL_TAG=<some tag> gh install ...` is equivalent to `gh install --download-regex <some regex> --tag <some tag>`

etc.
