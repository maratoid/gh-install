package cmd

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"runtime"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/maratoid/gh-install/release"
	"github.com/maratoid/gh-install/output"
	"github.com/spf13/cobra"
)

const (
	GH_INSTALL_VERSION_LATEST       = "latest"
	GH_INSTALL_ENV_PATH             = "GH_INSTALL_PATH"
	GH_INSTALL_CHECKSUM_ASSET_REGEX = ".*(?:checksum|txt)+.*$"
)

var (
	rootCmd = &cobra.Command{
		Use:   "gh install owner/repository",
		Short: "Install github release binaries",
		Long: `Install binaries for a Github repository release interactively or non-interactively.
			Intended for quickly installing release binaries for projects that do not distribute
			using Homebrew or other package managers.`,
		Args: validateRepositoryArg,
		RunE: runInstall,
		Version: "1.0.0",
	}
	targetRepo, releaseVersion, releaseInstallPath string
	downloadPattern, binaryPattern                 string
	interactive, jsonOut                           bool
	ghClient                                       *api.RESTClient
)

func validateRepositoryArg(cmd *cobra.Command, args []string) error {
	if jsonOut && !interactive {
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true
	}

	if len(args) != 1 {
		return fmt.Errorf("accepts %d arg(s), received %d", 1, len(args))
	}

	match, _ := regexp.MatchString(`.+/.+`, args[0])
	if !match {
		return fmt.Errorf("repository must be in 'user/repository' format (provided: %s)", args[0])
	}

	response := struct{ Name string }{}
	err := ghClient.Get(fmt.Sprintf("repos/%s", args[0]), &response)
	if err != nil {
		return fmt.Errorf("repository %s doesn't exist", args[0])
	}

	targetRepo = args[0]
	if binaryPattern == "" {
		binaryPattern = strings.Split(targetRepo, "/")[1]
	}

	return nil
}

func runInstall(cmd *cobra.Command, args []string) error {
	installRelease := release.MakeGithubRelease(
		targetRepo,
		releaseVersion,
		releaseInstallPath,
		downloadPattern,
		binaryPattern,
		ghClient,
		interactive)
	return installRelease.Install()
}

func Execute() {
	os.Exit(func () int {
		var err error
		ghClient, err = api.DefaultRESTClient()
		if err != nil {
			output.Output().Set("error", err.Error())
			printOutput(err)
			return 1
		}

		if err = rootCmd.Execute(); err != nil {
			output.Output().Set("error", err.Error())
			printOutput(err)
			return 1
		}


		printOutput(err)
		return 0
	}())
}

func printOutput(err error) {
	if interactive {
		return
	}

	if err == nil || jsonOut {
		output.Output().Print(jsonOut)
	}
}

func init() {
	rootCmd.Flags().StringVarP(&binaryPattern, "binaries", "b", "",
		"install release asset archive binaries matching pattern. If empty, repository name is used.")
	rootCmd.Flags().StringVarP(&releaseVersion, "tag", "t", GH_INSTALL_VERSION_LATEST,
		"release tag (version) to install.")
	rootCmd.Flags().StringVarP(&downloadPattern, "download", "d",
		fmt.Sprintf("^.*(?:%s.+%s|%s.+%s)+.*$", runtime.GOARCH, runtime.GOOS, runtime.GOOS, runtime.GOARCH),
		"name or lookup regexp for release asset to download.")
	rootCmd.Flags().BoolVarP(&jsonOut, "json", "j", false,
		"JSON output")
	rootCmd.Flags().BoolVarP(&interactive, "interactive", "i", false,
		"Use interactive installation. If true, all other flags are ignored")

	releaseInstallPath = os.Getenv(GH_INSTALL_ENV_PATH)
	if releaseInstallPath == "" {
		releaseInstallPath = path.Join(os.Getenv("HOME"), ".local", "bin")
	}
}
