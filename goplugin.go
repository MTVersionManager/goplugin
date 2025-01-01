package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"

	"github.com/MTVersionManager/mtvmplugin"
)

type Plugin struct {
	pw *progressWriter
}

type progressWriter struct {
	total           int
	downloaded      int
	Resp            *http.Response
	Content         []byte
	ProgressChannel chan float64
}

func (pw *progressWriter) Start() {
	_, err := io.Copy(pw, pw.Resp.Body)
	if err != nil {
		log.Fatal(err)
	}
}

// Compile time check to make sure that the plugin is actually a valid MTVM plugin
var _ mtvmplugin.Plugin = &Plugin{}

func (pw *progressWriter) Write(p []byte) (int, error) {
	pw.downloaded += len(p)
	pw.Content = append(pw.Content, p...)
	if pw.total > 0 {
		pw.ProgressChannel <- float64(pw.downloaded) / float64(pw.total)
	}
	return len(p), nil
}

func (p *Plugin) GetLatestVersion() (string, error) {
	resp, err := http.Get("https://go.dev/VERSION?m=text")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%v %v", resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil
	}
	bodystring := string(body)
	version := strings.Split(bodystring, "\n")[0][2:]
	return version, nil
}

func (p *Plugin) Download(version string, progress chan float64) error {
	url := createUrl(version)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%v %v", resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	if resp.ContentLength <= 0 {
		return errors.New("error when getting content length")
	}
	p.pw = &progressWriter{
		total:           int(resp.ContentLength),
		Resp:            resp,
		ProgressChannel: progress,
	}
	go p.pw.Start()
	return nil
}

func (p *Plugin) Install(installDir string) error {
	reader := bytes.NewReader(p.pw.Content)
	err := extractTarGZ(reader, installDir, rename)
	if err != nil {
		return err
	}
	err = p.pw.Resp.Body.Close()
	if err != nil {
		return err
	}
	return nil
}

func (p *Plugin) Use(installDir string, pathDir string) error {
	contents, err := os.ReadDir(filepath.Join(installDir, "bin"))
	if err != nil {
		return err
	}
	for _, file := range contents {
		if !file.IsDir() {
			linkPath := filepath.Join(pathDir, file.Name())
			_, err := os.Stat(linkPath)
			if err != nil && !os.IsNotExist(err) {
				return err
			} else if !os.IsNotExist(err) {
				// Delete old symlink if it exists
				err := os.Remove(linkPath)
				if err != nil {
					return err
				}
			}
			// Create new symlink
			err = os.Symlink(filepath.Join(installDir, "bin", file.Name()), linkPath)
			if err != nil {
				return err
			}
			// Give symlink execution permissions
			err = os.Chmod(linkPath, 0o755)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *Plugin) Remove(installDir string, pathDir string, inUse bool) error {
	if inUse {
		err := os.RemoveAll(filepath.Join(pathDir, "go") + BinaryExtension)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		err = os.RemoveAll(filepath.Join(pathDir, "gofmt") + BinaryExtension)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	err := os.RemoveAll(installDir)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (p *Plugin) Sort(stringVersions []string) ([]string, error) {
	var versions []*semver.Version
	for _, versionString := range stringVersions {
		version, err := semver.NewVersion(versionString)
		if err != nil {
			return nil, err
		}
		versions = append(versions, version)
	}
	sort.Sort(semver.Collection(versions))
	var sortedVersions []string
	for _, version := range versions {
		sortedVersions = append(sortedVersions, version.Original())
	}
	return sortedVersions, nil
}

func (p *Plugin) GetCurrentVersion(installDir string, pathDir string) (string, error) {
	linkSource, err := filepath.EvalSymlinks(filepath.Join(pathDir, "go") + BinaryExtension)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSuffix(strings.TrimPrefix(linkSource, installDir+string(os.PathSeparator)), string(os.PathSeparator)+"go"+BinaryExtension), nil
}

func rename(path string) string {
	return strings.TrimPrefix(path, "go"+string(os.PathSeparator))
}

func extractTarGZ(compressedStream io.Reader, directory string, renamer func(string) string) error {
	gzipReader, err := gzip.NewReader(compressedStream)
	if err != nil {
		return fmt.Errorf("gzip reader error: %w", err)
	}
	err = os.MkdirAll(directory, 0o777)
	if err != nil {
		return err
	}
	tarReader := tar.NewReader(gzipReader)
	var header *tar.Header
	for header, err = tarReader.Next(); err == nil; header, err = tarReader.Next() {
		renameResult := renamer(header.Name)
		if renameResult != "" && renameResult != string(os.PathSeparator) {
			joined := filepath.Join(directory, renameResult)
			switch header.Typeflag {
			case tar.TypeDir:
				if err := os.Mkdir(joined, 0o777); err != nil {
					return fmt.Errorf("ExtractTarGz: Mkdir() failed: %w", err)
				}
			case tar.TypeReg:
				outFile, err := os.Create(joined)
				if err != nil {
					return fmt.Errorf("ExtractTarGz: Create() failed: %w", err)
				}

				if _, err := io.Copy(outFile, tarReader); err != nil {
					// outFile.Close error omitted as Copy error is more interesting at this point
					outFile.Close()
					return fmt.Errorf("ExtractTarGz: Copy() failed: %w", err)
				}
				if err := outFile.Close(); err != nil {
					return fmt.Errorf("ExtractTarGz: Close() failed: %w", err)
				}
			default:
				return fmt.Errorf("ExtractTarGz: uknown type: %b in %s", header.Typeflag, header.Name)
			}
		}
	}
	if err != io.EOF {
		return fmt.Errorf("ExtractTarGz: Next() failed: %w", err)
	}
	return nil
}
