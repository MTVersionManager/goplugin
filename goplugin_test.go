package main

import (
	"os"
	"path/filepath"
	"testing"
)

type testDirs struct {
	InstallDir string
	PathDir    string
}

func TestSort(t *testing.T) {
	plugin := &Plugin{}
	unsortedVers := []string{"1.23.3", "1.20", "1.20.14", "1.22.4"}
	wantVers := []string{"1.20", "1.20.14", "1.22.4", "1.23.3"}
	sortedVers, err := plugin.Sort(unsortedVers)
	if err != nil {
		t.Fatal(err)
	}
	for i, ver := range sortedVers {
		if ver != wantVers[i] {
			t.Fatalf("want %s, got %s", wantVers, sortedVers)
		}
	}
}

func TestGetCurrentVersion(t *testing.T) {
	dirs, err := createTestDirs()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err = removeTestDirs(dirs)
		if err != nil {
			t.Fatal(err)
		}
	}()
	versionDir := filepath.Join(dirs.InstallDir, "1.23.3")
	err = os.MkdirAll(versionDir, 0o777)
	if err != nil {
		t.Fatal(err)
	}
	ogFilePath := filepath.Join(versionDir, "go"+BinaryExtension)
	file, err := os.Create(ogFilePath)
	if err != nil {
		t.Fatal(err)
	}
	err = file.Close()
	if err != nil {
		t.Fatal(err)
	}
	err = os.Symlink(ogFilePath, filepath.Join(dirs.PathDir, "go"+BinaryExtension))
	if err != nil {
		t.Fatal(err)
	}
	plugin := &Plugin{}
	ver, err := plugin.GetCurrentVersion(dirs.InstallDir, dirs.PathDir)
	if err != nil {
		t.Fatal(err)
	}
	if ver != "1.23.3" {
		t.Fatalf("want 1.23.3, got %s", ver)
	}
}

func TestGetCurrentVersionNotSet(t *testing.T) {
	dirs, err := createTestDirs()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err = removeTestDirs(dirs)
		if err != nil {
			t.Fatal(err)
		}
	}()
	plugin := &Plugin{}
	ver, err := plugin.GetCurrentVersion(dirs.InstallDir, dirs.PathDir)
	if err != nil {
		t.Fatal(err)
	}
	if ver != "" {
		t.Fatalf("want empty string, got %s", ver)
	}
}

func TestRemoveCurrentVer(t *testing.T) {
	dirs, err := createTestDirs()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err = removeTestDirs(dirs)
		if err != nil {
			t.Fatal(err)
		}
	}()
	ogFilePath := filepath.Join(dirs.InstallDir, "go"+BinaryExtension)
	file, err := os.Create(ogFilePath)
	if err != nil {
		t.Fatal(err)
	}
	err = file.Close()
	if err != nil {
		t.Fatal(err)
	}
	linkPath := filepath.Join(dirs.PathDir, "go"+BinaryExtension)
	err = os.Symlink(ogFilePath, linkPath)
	if err != nil {
		t.Fatal(err)
	}
	plugin := &Plugin{}
	err = plugin.Remove(dirs.InstallDir, dirs.PathDir, true)
	if err != nil {
		t.Fatal(err)
	}
	_, err = os.Stat(ogFilePath)
	if err == nil {
		t.Fatal("file should not exist")
	} else if !os.IsNotExist(err) {
		t.Fatal(err)
	}
	_, err = os.Stat(linkPath)
	if err == nil {
		t.Fatal("link should not exist")
	} else if !os.IsNotExist(err) {
		t.Fatal(err)
	}
}

func TestRemoveCurrentVerNotSet(t *testing.T) {
	dirs, err := createTestDirs()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err = removeTestDirs(dirs)
		if err != nil {
			t.Fatal(err)
		}
	}()
	ogFilePath := filepath.Join(dirs.InstallDir, "go"+BinaryExtension)
	file, err := os.Create(ogFilePath)
	if err != nil {
		t.Fatal(err)
	}
	err = file.Close()
	if err != nil {
		t.Fatal(err)
	}
	otherVerPath := filepath.Join(dirs.PathDir, "go"+BinaryExtension)
	file, err = os.Create(otherVerPath)
	if err != nil {
		t.Fatal(err)
	}
	err = file.Close()
	if err != nil {
		t.Fatal(err)
	}
	plugin := &Plugin{}
	err = plugin.Remove(dirs.InstallDir, dirs.PathDir, false)
	if err != nil {
		t.Fatal(err)
	}
	_, err = os.Stat(otherVerPath)
	if err != nil {
		if os.IsNotExist(err) {
			t.Fatal("other version should exist")
		}
		t.Fatal(err)
	}
	_, err = os.Stat(ogFilePath)
	if err == nil {
		t.Fatal("file should not exist")
	}
	if !os.IsNotExist(err) {
		t.Fatal(err)
	}
}

func TestRename(t *testing.T) {
	renamed := rename(filepath.Join("go", "someFile"))
	if renamed != "someFile" {
		t.Fatalf("want someFile, got %s", renamed)
	}
}

func TestProgressWriter_Write(t *testing.T) {
	pw := progressWriter{
		total:           100,
		ProgressChannel: make(chan float64),
	}
	returnDataChannel := make(chan struct {
		int
		error
	})
	dataToWrite := make([]byte, 50)
	go func() {
		written, err := pw.Write(dataToWrite)
		returnDataChannel <- struct {
			int
			error
		}{
			written,
			err,
		}
	}()
	for i := 0; i < 2; i++ {
		select {
		case returnedData := <-returnDataChannel:
			if returnedData.error != nil {
				t.Fatal(returnedData.error)
			}
			if returnedData.int != 50 {
				t.Fatalf("want 50 bytes written, got %v bytes written", returnedData.int)
			}
		case progress := <-pw.ProgressChannel:
			if progress != 0.5 {
				t.Errorf("want 0.5 progress, got %v progress", progress)
			}
		}
	}
	if len(pw.Content) != 50 {
		t.Fatalf("want 50 bytes of content, got %v bytes of content", len(pw.Content))
	}
}

func createTestDirs() (testDirs, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return testDirs{}, err
	}
	installDir := filepath.Join(homeDir, "testInstallDir")
	err = os.MkdirAll(installDir, 0o777)
	if err != nil {
		return testDirs{}, err
	}
	pathDir := filepath.Join(homeDir, "testPathDir")
	err = os.MkdirAll(pathDir, 0o777)
	if err != nil {
		return testDirs{}, err
	}
	return testDirs{
		InstallDir: installDir,
		PathDir:    pathDir,
	}, nil
}

func removeTestDirs(dirs testDirs) error {
	err := os.RemoveAll(dirs.InstallDir)
	if err != nil {
		return err
	}
	err = os.RemoveAll(dirs.PathDir)
	if err != nil {
		return err
	}
	return nil
}
