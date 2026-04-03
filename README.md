# gh-install
Github client extension for [installing Github repository releases](https://maratg.com/posts/gh-install-extension/)

## Installation

```
 gh extension install maratoid/gh-install
```

## Help

```
$ gh install --help
Usage: gh-install <repository> [flags]

Install binaries for a Github repository release interactively or non-interactively.

    Intended for quickly installing release binaries for projects that do not distribute
    using Homebrew or other package managers.

Arguments:
  <repository>    Github repository in OWNER/REPOSITORY_NAME format ($GH_INSTALL_REPOSITORY).

Flags:
  -h, --help                                        Show context-sensitive help.
  -p, --target-path="/Users/maratoid/.local/bin"    Target installation directory ($GH_INSTALL_TARGET_PATH).
  -l, --log-level="info"                            Log level ($GH_INSTALL_LOG_LEVEL).
  -f, --log-format="console"                        Log output format ($GH_INSTALL_LOG_FORMAT).
      --version                                     Show version ($GH_INSTALL_VERSION).

Interactive Mode
  -i, --interactive                   Use interactive installation. If true, all non-log related flags are ignored ($GH_INSTALL_INTERACTIVE).
      --[no-]log-quiet-interactive    Quiet log in interactive mode ($GH_INSTALL_LOG_QUIET_INTERACTIVE)

Non-interactive Mode
  -v, --release-version="latest"                                         Repository release tag (version) to install ($GH_INSTALL_RELEASE_VERSION).
  -a, --release-asset=STRING                                             Name of repository release asset to download. If not set, --release-asset-regexp is used ($GH_INSTALL_RELEASE_ASSET).
  -A, --release-asset-regexp="^.*(?:arm64.+darwin|darwin.+arm64)+.*$"    Regular expression matching release asset to download ($GH_INSTALL_RELEASE_ASSET_REGEXP).
  -b, --asset-binaries=ASSET-BINARIES,...                                If release asset is an archive - names of a binaries in the archive to install. If not set, --install-binary-regexp is used ($GH_INSTALL_ASSET_BINARIES).
  -B, --asset-binaries-regexp=STRING                                     If release asset is an archive - regular expression matching binaries in the archive to install. If not set, repository name is used ($GH_INSTALL_ASSET_BINARIES_REGEXP).
  -t, --target-binaries=KEY=VALUE;...                                    Rename binaries installed at target path, "<asset archive binary | asset>=<renamed binary>;..." ($GH_INSTALL_TARGET_BINARIES)
      --[no-]target-path-create                                          Create target installation directory if it does not exist ($GH_INSTALL_TARGET_PATH_CREATE).
  -o, --target-binaries-overwrite                                        Overwrite target binaries ($GH_INSTALL_TARGET_BINARIES_OVERWRITE).
```

Alternatively, set parameter values via corresponding environment variables. E.g.:

* `GH_INSTALL_REPOSITORY=charmbracelet/gum gh install -i` is equivalent to `gh install charmbracelet/gum -i`
* `GH_INSTALL_RELEASE_ASSET=gum_0.17.0_Darwin_arm64.tar.gz gh install charmbracelet/gum` is equivalent to `gh install charmbracelet/gum --release-asset gum_0.17.0_Darwin_arm64.tar.gz`

etc.

`GH_INSTALL` prefix itself is configurable via `GH_INSTALL_ENV_PREFIX` environment variable:

```
$ GH_INSTALL_ENV_PREFIX=YEET ./gh-install -h
Usage: gh-install <repository> [flags]

Install binaries for a Github repository release interactively or non-interactively.

    Intended for quickly installing release binaries for projects that do not distribute
    using Homebrew or other package managers.

Arguments:
  <repository>    Github repository in OWNER/REPOSITORY_NAME format ($YEET_REPOSITORY).
...
```
