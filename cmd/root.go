package cmd

import (
	"errors"
	"fmt"
	"os"
	"path"
	"regexp"
	"runtime"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/maratoid/gh-install/params"
	"github.com/maratoid/gh-install/release"
	"github.com/pterm/pterm"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	GH_INSTALL_PREFIX_ENV           = "GH_INSTALL_ENV_PREFIX"
	GH_INSTALL_DEFAULT_PREFIX       = "GH_INSTALL"
	GH_INSTALL_CHECKSUM_ASSET_REGEX = ".*(?:checksum|txt)+.*$"
)

type RootCLI params.CLI

func (r *RootCLI) Validate() error {
	match, _ := regexp.MatchString(`.+/.+`, r.Repository)
	if !match {
		return fmt.Errorf("repository must be in 'user/repository' format (provided: '%s')", r.Repository)
	}

	if r.TargetPath == "" {
		err := fmt.Errorf("could not determine default install path, use '--install-path' flag")
		log.Error().
			Err(err).
			Msg("init error")
		return err
	}

	targetPathInfo, err := os.Stat(r.TargetPath)
	if err != nil {
		if !os.IsNotExist(err) {
			createPath := r.TargetPathCreate
			if r.Interactive {
				createPath, _ = pterm.DefaultInteractiveConfirm.
					WithDefaultValue(true).
					Show(fmt.Sprintf("'%s' does not exist. Create?", r.TargetPath))
			}

			if createPath {
				err := os.MkdirAll(r.TargetPath, os.ModePerm)
				if err != nil {
					log.Error().
						Err(err).
						Msgf("target installation path '%s' error", r.TargetPath)
					return err
				}
				return nil
			} else {
				log.Error().
					Err(err).
					Msgf("target installation path '%s' error", r.TargetPath)
				return err
			}

		}
		log.Error().
			Err(err).
			Msgf("target installation path '%s' error", r.TargetPath)
		return err
	}

	if !targetPathInfo.Mode().IsDir() {
		err = errors.New("not a directory")
		log.Error().
			Err(err).
			Msgf("target installation path '%s' error", r.TargetPath)
	}

	return nil
}

func PostBuild(k *kong.Kong) error {
	k.Model.Positional[0].Tag.Envs = []string{fmt.Sprintf("%s_REPOSITORY", GetEnvPrefix())}
	return nil
}

func (r *RootCLI) Run() error {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	logLevel, _ := zerolog.ParseLevel(r.LogLevel)
	if r.LogQuietInteractive && r.Interactive {
		logLevel = zerolog.Disabled
	}
	zerolog.SetGlobalLevel(logLevel)
	if r.LogFormat == "console" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	}

	if r.AssetBinariesRegexp == "" {
		r.AssetBinariesRegexp = fmt.Sprintf("^%s$", strings.Split(r.Repository, "/")[1])
	}

	log.Info().
		Str("repository", r.Repository).
		Str("release version", r.ReleaseVersion).
		Str("release asset name", r.ReleaseAsset).
		Str("release asset regexp", r.ReleaseAssetRegexp).
		Array("release asset binary names", func() *zerolog.Array {
			arr := zerolog.Arr()
			for _, i := range r.AssetBinaries {
				arr = arr.Str(i)
			}
			return arr
		}()).
		Str("release asset binary name regexp", r.AssetBinariesRegexp).
		Str("target path", r.TargetPath).
		Dict("renaming binaries", func() *zerolog.Event {
			d := zerolog.Dict()
			for k, v := range r.TargetBinaries {
				d = d.Str(k, v)
			}
			return d
		}()).
		Msg("installing with values")

	ghClient, err := api.DefaultRESTClient()
	if err != nil {
		log.Error().
			Err(err).
			Msg("could not init Gihub REST client")
		return err
	}

	response := struct{ Name string }{}
	err = ghClient.Get(fmt.Sprintf("repos/%s", r.Repository), &response)
	if err != nil {
		log.Error().
			Err(err).
			Msgf("repository %s doesn't exist", r.Repository)
	}

	installRelease := release.MakeGithubRelease(
		(*params.CLI)(r),
		ghClient)
	return installRelease.Install()
}

func GetDefaultTargetPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return path.Join(homeDir, ".local", "bin")
}

func GetDefaultAssetRegexp() string {
	return fmt.Sprintf("^.*(?:%s.+%s|%s.+%s)+.*$", runtime.GOARCH, runtime.GOOS, runtime.GOOS, runtime.GOARCH)
}

func GetEnvPrefix() string {
	envPrefix := os.Getenv(GH_INSTALL_PREFIX_ENV)
	if envPrefix == "" {
		envPrefix = GH_INSTALL_DEFAULT_PREFIX
	}

	return strings.ToUpper(envPrefix)
}
