package cmd

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"runtime"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/maratoid/gh-install/output"
	"github.com/maratoid/gh-install/release"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
		Args:    validateRepositoryArg,
		RunE:    runInstall,
		Version: "1.1.2",
	}
	targetRepo, releaseVersion, releaseInstallPath           string
	downloadPattern, binaryPattern, binaryName, downloadName string
	interactive, jsonOut, noCreatePath                       bool
	ghClient                                                 *api.RESTClient
)

func validateRepositoryArg(cmd *cobra.Command, args []string) error {
	if viper.GetBool("json") && !viper.GetBool("interactive") {
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
	if viper.GetString("binary-regex") == "" {
		viper.Set("binary-regex", fmt.Sprintf("^%s$", strings.Split(targetRepo, "/")[1]))
	}

	return nil
}

func runInstall(cmd *cobra.Command, args []string) error {

	if viper.GetString("path") == "" {
		return fmt.Errorf("could not determine default installation directory, provide via '--path'")
	}

	if noCreatePath {
		targetPathInfo, err := os.Stat(viper.GetString("path"))
		if err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("target installation path '%s' does not exist", viper.GetString("path"))
			}
			return fmt.Errorf("could not stat %s: %v", viper.GetString("path"), err)
		}

		if !targetPathInfo.Mode().IsDir() {
			return fmt.Errorf("target installation path '%s' is not a directory", viper.GetString("path"))
		}
	} else {
		err := os.MkdirAll(viper.GetString("path"), os.ModePerm)
		if err != nil {
			return err
		}
	}

	installRelease := release.MakeGithubRelease(
		targetRepo,
		viper.GetString("tag"),
		viper.GetString("path"),
		viper.GetString("download"),
		viper.GetString("download-regex"),
		viper.GetString("binary"),
		viper.GetString("binary-regex"),
		ghClient,
		viper.GetBool("interactive"))
	return installRelease.Install()
}

func Execute() {
	os.Exit(func() int {
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
	if viper.GetBool("interactive") {
		return
	}

	if err == nil || jsonOut {
		output.Output().Print(jsonOut)
	}
}

func getDefaultInstallPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return path.Join(homeDir, ".local", "bin")
}

func init() {
	viper.SetEnvPrefix("gh_install")
	rootCmd.Flags().StringVarP(&binaryName, "binary", "b", "",
		"install release asset archive binary name. If empty, '--binary-regex' is used.")
	rootCmd.Flags().StringVarP(&binaryPattern, "binary-regex", "", "",
		"lookup regexp for release asset archive binary. If empty, repository name is used.")
	rootCmd.Flags().StringVarP(&releaseVersion, "tag", "t", GH_INSTALL_VERSION_LATEST,
		"release tag (version) to install.")
	rootCmd.Flags().StringVarP(&downloadName, "download", "d", "",
		"name for release asset to download. If empty, '--download-regex' is used.")
	rootCmd.Flags().StringVarP(&downloadPattern, "download-regex", "",
		fmt.Sprintf("^.*(?:%s.+%s|%s.+%s)+.*$", runtime.GOARCH, runtime.GOOS, runtime.GOOS, runtime.GOARCH),
		"lookup regexp for release asset to download.")
	rootCmd.Flags().BoolVarP(&jsonOut, "json", "j", false,
		"JSON output")
	rootCmd.Flags().BoolVarP(&interactive, "interactive", "i", false,
		"Use interactive installation. If true, all other flags are ignored")
	rootCmd.Flags().StringVarP(&releaseInstallPath, "path", "p", getDefaultInstallPath(),
		"Target installation directory.")
	rootCmd.Flags().BoolVarP(&noCreatePath, "no-create", "", false,
		"Do not create target installation directory if it does not exist.")
	viper.BindPFlags(rootCmd.Flags())
	viper.AutomaticEnv()
}
