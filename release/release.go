package release

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"

	"github.com/cli/go-gh/v2"
	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/maratoid/gh-install/output"
	"github.com/maratoid/gh-install/selector"
)

type IRelease interface {
	Install() error
}

type GithubRelease struct {
	Repository         string
	Interactive        bool
	ReleaseVersion     string
	InstallPath        string
	AssetName          string
	AssetPattern       string
	AssetBinaryName    string
	AssetBinaryPattern string
	Client             *api.RESTClient
}

func MakeGithubRelease(repo string, ver string, dest string,
	assetName string, assetMatcher string, binName string, binMatcher string,
	cli *api.RESTClient, interactive bool) IRelease {

	return &GithubRelease{
		Repository:         repo,
		Interactive:        interactive,
		ReleaseVersion:     ver,
		InstallPath:        dest,
		AssetName:          assetName,
		AssetPattern:       assetMatcher,
		AssetBinaryName:    binName,
		AssetBinaryPattern: binMatcher,
		Client:             cli,
	}
}

func (r *GithubRelease) installArchivedBinary(fileSystem fs.FS, binaryPath string) error {
	sourceFile, err := fileSystem.Open(binaryPath)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destinationPath := path.Join(r.InstallPath, path.Base(binaryPath))
	destinationFile, err := os.Create(destinationPath)
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		return err
	}

	err = os.Chmod(destinationPath, 0755)
	if err != nil {
		return err
	}

	return nil
}

func (r *GithubRelease) installBinary(binaryPath string) error {
	sourceStat, err := os.Stat(binaryPath)
	if err != nil {
		return err
	}

	if !sourceStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", binaryPath)
	}

	source, err := os.Open(binaryPath)
	if err != nil {
		return err
	}
	defer source.Close()

	destinationPath := path.Join(r.InstallPath, path.Base(binaryPath))
	destination, err := os.Create(destinationPath)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	if err != nil {
		return err
	}

	err = os.Chmod(destinationPath, 0755)
	if err != nil {
		return err
	}

	return nil
}

func (r *GithubRelease) installDeb(binaryPath string) error {
	cmd := exec.Command("dpkg", "-i", binaryPath)
	_, err := cmd.Output()
	if err != nil {
		return err
	}
	return nil
}

func (r *GithubRelease) installRpm(binaryPath string) error {
	cmd := exec.Command("dnf", "localinstall", binaryPath)
	_, err := cmd.Output()
	if err != nil {
		return err
	}
	return nil
}

func (r *GithubRelease) Install() error {
	var err error
	defer func() {
		if err != nil {
			output.Output().Set("error", err)
		}
	}()
	output.Output().Set("target_repository", r.Repository)
	output.Output().Set("install_dir", r.InstallPath)

	releaseSelector, err := selector.ReleaseSelector(r.Client, r.Repository, r.ReleaseVersion, r.Interactive)
	if err != nil {
		return err
	}
	releases, err := releaseSelector.SelectItems()
	if err != nil {
		return err
	}

	assetSelector, err := selector.AssetSelector(r.Client, r.Repository, releases[0].GetPropInt("id"), r.AssetName, r.AssetPattern, r.Interactive)
	if err != nil {
		return err
	}
	assets, err := assetSelector.SelectItems()
	if err != nil {
		return err
	}

	downloadDir, err := os.MkdirTemp("", "*")
	if err != nil {
		return err
	}
	output.Output().Set("download_dir", downloadDir)

	stdOut, stdErr, err := gh.Exec("release", "download", releases[0].Name,
		"--repo", r.Repository, "--pattern", assets[0].Name, "--dir", downloadDir)
	if err != nil {
		output.Output().Set("gh_stderr", stdErr.String())
		return fmt.Errorf("failed to run gh command: %s", stdErr.String())
	}
	output.Output().Set("gh_stdout", stdOut.String())

	binarySelector, err := selector.BinarySelector(path.Join(downloadDir,
		assets[0].Name), r.AssetBinaryName, r.AssetBinaryPattern, r.Interactive)
	if err != nil {
		return err
	}
	binaries, err := binarySelector.SelectItems()
	if err != nil {
		return err
	}

	binariesOutput := make(map[string]string)
	for _, binary := range binaries {
		if binary.GetPropBool("archive") {
			binariesOutput[binary.Name] = "compressed"
			err = r.installArchivedBinary(binary.GetPropFs("fs"), binary.GetPropStr("path"))
		} else {
			if binary.GetPropStr("binType") == "deb" {
				binariesOutput[binary.Name] = "deb"
				err = r.installDeb(binary.GetPropStr("path"))
			} else if binary.GetPropStr("binType") == "rpm" {
				binariesOutput[binary.Name] = "rpm"
				err = r.installRpm(binary.GetPropStr("path"))
			} else {
				binariesOutput[binary.Name] = "binary"
				err = r.installBinary(binary.GetPropStr("path"))
			}
		}
		if err != nil {
			output.Output().Set("asset_installed_binaries", binariesOutput)
			return err
		}
	}

	output.Output().Set("asset_installed_binaries", binariesOutput)
	return nil
}
