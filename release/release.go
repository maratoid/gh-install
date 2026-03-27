package release

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/cli/go-gh/v2"
	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/maratoid/gh-install/params"
	"github.com/maratoid/gh-install/selector"
	"github.com/pterm/pterm"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type IRelease interface {
	Install() error
}

type GithubRelease struct {
	CliParams *params.CLI
	Client    *api.RESTClient
}

func MakeGithubRelease(cliParams *params.CLI, cli *api.RESTClient) IRelease {

	return &GithubRelease{
		CliParams: cliParams,
		Client:    cli,
	}
}

func (r *GithubRelease) interactiveConfirm(prompt string) bool {
	result, _ := pterm.DefaultInteractiveConfirm.
		WithDefaultValue(true).
		Show(prompt)
	return result
}

func (r *GithubRelease) interactiveInput(prompt string, defaultValue string) string {
	result, _ := pterm.DefaultInteractiveTextInput.
		WithDefaultValue(defaultValue).
		Show(prompt)
	return result
}

func (r *GithubRelease) installArchivedBinary(fileSystem fs.FS, binaryPath string) error {
	sourceFile, err := fileSystem.Open(binaryPath)
	if err != nil {
		return err
	}

	defer func() {
		err = errors.Join(err, sourceFile.Close())
	}()

	binaryName := path.Base(binaryPath)
	destinationPath := path.Join(r.CliParams.TargetPath, binaryName)
	if targetBinaryName, exists := r.CliParams.TargetBinaries[strings.ToLower(binaryName)]; exists {
		destinationPath = path.Join(r.CliParams.TargetPath, targetBinaryName)
	}

	if r.CliParams.Interactive {
		destinationPath = r.interactiveInput(fmt.Sprintf("Install %s to", binaryPath), destinationPath)
	}

	log.Info().
		Msgf("will install %s to %s", binaryPath, destinationPath)

	_, err = os.Stat(destinationPath)
	if err == nil {
		if r.CliParams.Interactive {
			if !r.interactiveConfirm(fmt.Sprintf("'%s' already exists. Overwrite?", destinationPath)) {
				return fmt.Errorf("%s already exists and user did not want to overwrite", destinationPath)
			}
		} else {
			if !r.CliParams.TargetBinariesOverwrite {
				return fmt.Errorf("%s already exists and --target-binaries-overwrite is not set", destinationPath)
			}
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	destinationFile, err := os.Create(destinationPath)
	if err != nil {
		return err
	}

	defer func() {
		err = errors.Join(err, destinationFile.Close())
	}()

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

	defer func() {
		err = errors.Join(err, source.Close())
	}()

	binaryName := path.Base(binaryPath)
	destinationPath := path.Join(r.CliParams.TargetPath, binaryName)
	if targetBinaryName, exists := r.CliParams.TargetBinaries[strings.ToLower(binaryName)]; exists {
		destinationPath = path.Join(r.CliParams.TargetPath, targetBinaryName)
	}

	if r.CliParams.Interactive {
		destinationPath = r.interactiveInput(fmt.Sprintf("Install %s to", binaryPath), destinationPath)
	}

	log.Info().
		Msgf("will install %s to %s", binaryPath, destinationPath)

	_, err = os.Stat(destinationPath)
	if err == nil {
		if r.CliParams.Interactive {
			if !r.interactiveConfirm(fmt.Sprintf("'%s' already exists. Overwrite?", destinationPath)) {
				return fmt.Errorf("%s already exists and user did not want to overwrite", destinationPath)
			}
		} else {
			if !r.CliParams.TargetBinariesOverwrite {
				return fmt.Errorf("%s already exists and --target-binaries-overwrite is not set", destinationPath)
			}
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	destination, err := os.Create(destinationPath)
	if err != nil {
		return err
	}

	defer func() {
		err = errors.Join(err, destination.Close())
	}()

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
	var installSpinner *pterm.SpinnerPrinter
	if r.CliParams.Interactive {
		if !r.interactiveConfirm(fmt.Sprintf("Run 'dpkg -i %s'?", binaryPath)) {
			return fmt.Errorf("'%s' is a DEB installer and user did not want to run it", binaryPath)
		}
		installSpinner, _ = pterm.DefaultSpinner.Start(fmt.Sprintf("Running 'dpkg -i %s'...", binaryPath))
	}

	cmd := exec.Command("dpkg", "-i", binaryPath)
	out, err := cmd.Output()
	if err != nil {
		var errOut string
		if ee, ok := err.(*exec.ExitError); ok {
			errOut = string(ee.Stderr)
		}
		log.Error().
			Str("installer binary", binaryPath).
			Str("installer output", errOut).
			Err(err).
			Msgf("'dpkg -i %s' failed", binaryPath)
		if r.CliParams.Interactive {
			installSpinner.Fail("Failed.")
		}
		return err
	}
	log.Info().
		Str("installer binary", binaryPath).
		Str("installer output", string(out)).
		Msgf("ran 'dpkg -i %s'", binaryPath)
	if r.CliParams.Interactive {
		installSpinner.Fail("Success!")
	}
	return nil
}

func (r *GithubRelease) installRpm(binaryPath string) error {
	var installSpinner *pterm.SpinnerPrinter
	if r.CliParams.Interactive {
		if !r.interactiveConfirm(fmt.Sprintf("Run 'dnf localinstall %s'?", binaryPath)) {
			return fmt.Errorf("'%s' is a RPM installer and user did not want to run it", binaryPath)
		}
		installSpinner, _ = pterm.DefaultSpinner.Start(fmt.Sprintf("Running 'dnf localinstall %s'...", binaryPath))
	}
	cmd := exec.Command("dnf", "localinstall", binaryPath)
	out, err := cmd.Output()
	if err != nil {
		var errOut string
		if ee, ok := err.(*exec.ExitError); ok {
			errOut = string(ee.Stderr)
		}
		log.Error().
			Str("installer binary", binaryPath).
			Str("installer output", errOut).
			Err(err).
			Msgf("'dnf localinstall %s' failed", binaryPath)
		if r.CliParams.Interactive {
			installSpinner.Fail("Failed.")
		}
		return err
	}
	log.Info().
		Str("installer binary", binaryPath).
		Str("installer output", string(out)).
		Msgf("ran 'dnf localinstall %s'", binaryPath)
	if r.CliParams.Interactive {
		installSpinner.Fail("Success!")
	}
	return nil
}

func (r *GithubRelease) Install() error {
	releaseSelector, err := selector.ReleaseSelector(r.Client, r.CliParams.Repository, r.CliParams.ReleaseVersion, r.CliParams.Interactive)
	if err != nil {
		log.Error().
			Str("repository", r.CliParams.Repository).
			Str("release version", r.CliParams.ReleaseVersion).
			Err(err).
			Msg("could not create release selector")
		return err
	}
	releases, err := releaseSelector.Run()
	if err != nil {
		log.Error().
			Str("repository", r.CliParams.Repository).
			Str("release version", r.CliParams.ReleaseVersion).
			Err(err).
			Msg("could not select a release")
		return err
	}

	assetSelector, err := selector.AssetSelector(r.Client, r.CliParams.Repository, releases[0].GetId(),
		r.CliParams.ReleaseAsset, r.CliParams.ReleaseAssetRegexp, r.CliParams.Interactive)
	if err != nil {
		log.Error().
			Str("repository", r.CliParams.Repository).
			Int("release id", releases[0].GetId()).
			Str("release name", releases[0].Name).
			Str("asset name matcher", r.CliParams.ReleaseAsset).
			Str("asset regexp matcher", r.CliParams.ReleaseAssetRegexp).
			Err(err).
			Msg("could not create release asset selector")
		return err
	}
	assets, err := assetSelector.Run()
	if err != nil {
		log.Error().
			Str("repository", r.CliParams.Repository).
			Int("release id", releases[0].GetId()).
			Str("release asset name matcher", r.CliParams.ReleaseAsset).
			Str("release asset regexp matcher", r.CliParams.ReleaseAssetRegexp).
			Err(err).
			Msg("could not select release asset")
		return err
	}

	var downloadSpinner *pterm.SpinnerPrinter
	if r.CliParams.Interactive {
		downloadSpinner, _ = pterm.DefaultSpinner.Start(fmt.Sprintf("Downloading asset '%s'...", assets[0].Name))
	}

	downloadDir, err := os.MkdirTemp("", "*")
	if err != nil {
		log.Error().
			Err(err).
			Msg("could not create temporary download directory")
		if r.CliParams.Interactive {
			downloadSpinner.Fail("Failed - could not create temporary download directory")
		}
		return err
	}

	stdOut, stdErr, err := gh.Exec("release", "download", releases[0].Name,
		"--repo", r.CliParams.Repository, "--pattern", assets[0].Name, "--dir", downloadDir)
	if err != nil {
		err = fmt.Errorf("failed to run gh command: %s", stdErr.String())
		log.Error().
			Str("repository", r.CliParams.Repository).
			Int("release id", releases[0].GetId()).
			Str("release name", releases[0].Name).
			Str("release asset name", assets[0].Name).
			Str("download directory", downloadDir).
			Err(err).
			Msg("could not download release asset")
		if r.CliParams.Interactive {
			downloadSpinner.Fail(fmt.Sprintf("Failed - '%s' failed", stdErr.String()))
		}
		return err
	}

	if r.CliParams.Interactive {
		downloadSpinner.Success("Downloaded!")
	}

	defer func() {
		err = errors.Join(err, os.RemoveAll(downloadDir))
	}()

	log.Info().
		Str("repository", r.CliParams.Repository).
		Int("release id", releases[0].GetId()).
		Str("release name", releases[0].Name).
		Str("release asset name", assets[0].Name).
		Str("download directory", downloadDir).
		Str("output", stdOut.String()).
		Msg("downloaded release asset")

	binarySelector, err := selector.BinarySelector(path.Join(downloadDir,
		assets[0].Name), r.CliParams.AssetBinaries, r.CliParams.AssetBinariesRegexp, r.CliParams.Interactive)
	if err != nil {
		log.Error().
			Str("repository", r.CliParams.Repository).
			Int("release id", releases[0].GetId()).
			Str("release name", releases[0].Name).
			Str("release asset name", assets[0].Name).
			Str("downloaded asset", path.Join(downloadDir, assets[0].Name)).
			Array("asset binary name matchers", func() *zerolog.Array {
				arr := zerolog.Arr()
				for _, i := range r.CliParams.AssetBinaries {
					arr = arr.Str(i)
				}
				return arr
			}()).
			Str("asset binary regexp matcher", r.CliParams.AssetBinariesRegexp).
			Err(err).
			Msg("could not create release asset binary selector")
		return err
	}
	binaries, err := binarySelector.Run()
	if err != nil {
		log.Error().
			Str("repository", r.CliParams.Repository).
			Int("release id", releases[0].GetId()).
			Str("release name", releases[0].Name).
			Str("release asset name", assets[0].Name).
			Str("downloaded asset", path.Join(downloadDir, assets[0].Name)).
			Array("asset binary name matchers", func() *zerolog.Array {
				arr := zerolog.Arr()
				for _, i := range r.CliParams.AssetBinaries {
					arr = arr.Str(i)
				}
				return arr
			}()).
			Str("asset binary regexp matcher", r.CliParams.AssetBinariesRegexp).
			Err(err).
			Msg("could not select release asset binary")
		return err
	}

	binariesOutput := make(map[string]string)
	for _, binary := range binaries {
		log.Info().
			Str("repository", r.CliParams.Repository).
			Int("release id", releases[0].GetId()).
			Str("release name", releases[0].Name).
			Str("release asset name", assets[0].Name).
			Str("downloaded asset", path.Join(downloadDir, assets[0].Name)).
			Str("release asset binary", binary.Name).
			Msg("processing selected release asset binary")
		if binary.GetCompressed() {
			log.Debug().
				Str("release asset binary", binary.Name).
				Str("release asset binary archive path", binary.GetFsPath()).
				Msg("binary is part of an archive")
			binariesOutput[binary.Name] = "compressed"
			err = r.installArchivedBinary(binary.GetFs(), binary.GetFsPath())
		} else {
			switch binary.GetBinaryType() {
			case selector.BinaryDebInstaller:
				log.Debug().
					Str("release asset binary", binary.Name).
					Msg("binary is a deb installer")
				binariesOutput[binary.Name] = "deb"
				err = r.installDeb(binary.GetDownloadPath())
			case selector.BinaryRpmInstaller:
				log.Debug().
					Str("release asset binary", binary.Name).
					Msg("binary is a rpm installer")
				binariesOutput[binary.Name] = "rpm"
				err = r.installRpm(binary.GetDownloadPath())
			default:
				log.Debug().
					Str("release asset binary", binary.Name).
					Msg("binary is a plain executable")
				binariesOutput[binary.Name] = "binary"
				err = r.installBinary(binary.GetDownloadPath())
			}
		}
		if err != nil {
			log.Error().
				Str("repository", r.CliParams.Repository).
				Int("release id", releases[0].GetId()).
				Str("release name", releases[0].Name).
				Str("release asset name", assets[0].Name).
				Str("downloaded asset", path.Join(downloadDir, assets[0].Name)).
				Str("release asset binary", binary.Name).
				Err(err).
				Msg("could not install release asset binary")
			return err
		}
	}
	return nil
}
