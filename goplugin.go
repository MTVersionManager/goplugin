package goplugin

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
	"strings"
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

func (pw *progressWriter) Write(p []byte) (int, error) {
	pw.downloaded += len(p)
	pw.Content = append(pw.Content, p...)
	if pw.total > 0 {
		fmt.Println("Updating progress and sending message!")
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

func (p *Plugin) Progress() *chan float64 {
	p.pw = new(progressWriter)
	p.pw.ProgressChannel = make(chan float64)
	return &p.pw.ProgressChannel
}

func (p *Plugin) Download(version string) error {
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
		total: int(resp.ContentLength),
		Resp:  resp,
	}
	go p.pw.Start()
	return nil
}

func (p *Plugin) Install(installDir string) error {
	err := p.pw.Resp.Body.Close()
	if err != nil {
		return err
	}
	reader := bytes.NewReader(p.pw.Content)
	err = ExtractTarGZ(reader, installDir, renamer)
	if err != nil {
		return err
	}
	return nil
}

func renamer(path string) string {
	if !strings.HasPrefix(path, "go"+string(os.PathSeparator)+"bin") {
		return ""
	}
	return strings.TrimPrefix(path, "go"+string(os.PathSeparator)+"bin")
}

func ExtractTarGZ(compressedStream io.Reader, directory string, renamer func(string) string) error {
	gzipReader, err := gzip.NewReader(compressedStream)
	if err != nil {
		return fmt.Errorf("gzip reader error: %w", err)
	}
	err = os.MkdirAll(directory, 0755)
	if err != nil {
		return err
	}
	tarReader := tar.NewReader(gzipReader)
	var header *tar.Header
	for header, err = tarReader.Next(); err == nil; header, err = tarReader.Next() {
		path := filepath.Join(directory, renamer(header.Name))
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.Mkdir(path, 0755); err != nil {
				return fmt.Errorf("ExtractTarGz: Mkdir() failed: %w", err)
			}
		case tar.TypeReg:
			outFile, err := os.Create(path)
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
	if err != io.EOF {
		return fmt.Errorf("ExtractTarGz: Next() failed: %w", err)
	}
	return nil
}
