package release

import (
	"os"
	"os/exec"
	"io"
	"io/fs"
	"path"
	"fmt"

	"github.com/cli/go-gh/v2"
	"github.com/cli/go-gh/v2/pkg/api"
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
	AssetPattern       string
	AssetBinaryPattern string
	Client             *api.RESTClient
}

func MakeGithubRelease(repo string, ver string, dest string, 
	assetMatcher string, binMatcher string, cli *api.RESTClient, interactive bool) IRelease {
	
	return &GithubRelease{
		Repository: repo,
		Interactive: interactive,
		ReleaseVersion: ver,
		InstallPath: dest,
		AssetPattern: assetMatcher,
		AssetBinaryPattern: binMatcher,
		Client: cli,
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
	releaseSelector, err := selector.ReleaseSelector(r.Client, r.Repository, r.ReleaseVersion, r.Interactive)
	if err != nil {
		return err
	}
	releases, err := releaseSelector.SelectItems()
	if err != nil {
		return err
	}

	assetSelector, err := selector.AssetSelector(r.Client, r.Repository, releases[0].GetPropInt("id"), r.AssetPattern, r.Interactive)
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

	_, stdErr, err := gh.Exec("release", "download", releases[0].Name, 
		"--repo", r.Repository, "--pattern", assets[0].Name, "--dir", downloadDir)
	if err != nil {
		return fmt.Errorf("failed to run gh command: %s", stdErr.String())
	}


	binarySelector, err := selector.BinarySelector(path.Join(downloadDir, 
		assets[0].Name), r.AssetBinaryPattern, r.Interactive)
	if err != nil {
		return err
	}
	binaries, err := binarySelector.SelectItems()
	if err != nil {
		return err
	}

	var installedBinaries []string
	for _, binary := range binaries {
		if binary.GetPropBool("archive") {
			err = r.installArchivedBinary(binary.GetPropFs("fs"), binary.GetPropStr("path"))
		} else {
			if binary.GetPropStr("binType") == "deb" {
				err = r.installDeb(binary.GetPropStr("path"))
			} else if binary.GetPropStr("binType") == "rpm" {
				err = r.installRpm(binary.GetPropStr("path"))
			} else {
				err = r.installBinary(binary.GetPropStr("path"))
			}
		}
		if err != nil {
			return err
		}
		installedBinaries = append(installedBinaries, binary.Name)
	}
	return nil
}
