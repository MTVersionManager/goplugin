package goplugin

import (
	"os"
	"path/filepath"
	"testing"
)

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
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	installDir := filepath.Join(homeDir, "testInstallDir")
	defer func() {
		err = os.RemoveAll(installDir)
		if err != nil {
			t.Fatal(err)
		}
	}()
	versionDir := filepath.Join(installDir, "1.23.3")
	err = os.MkdirAll(versionDir, 0755)
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
	pathDir := filepath.Join(homeDir, "testPathDir")
	defer func() {
		err = os.RemoveAll(pathDir)
		if err != nil {
			t.Fatal(err)
		}
	}()
	err = os.MkdirAll(pathDir, 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.Symlink(ogFilePath, filepath.Join(pathDir, "go"+BinaryExtension))
	if err != nil {
		t.Fatal(err)
	}
	plugin := &Plugin{}
	ver, err := plugin.GetCurrentVersion(installDir, pathDir)
	if err != nil {
		t.Fatal(err)
	}
	if ver != "1.23.3" {
		t.Fatalf("want 1.23.3, got %s", ver)
	}
}
