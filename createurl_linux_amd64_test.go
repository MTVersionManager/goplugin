//go:build linux && amd64

package goplugin

import "testing"

func TestCreateUrl(t *testing.T) {
	url := createUrl("1.23.3")
	if url != "https://go.dev/dl/go1.23.3.linux-amd64.tar.gz" {
		t.Fatalf("want https://go.dev/dl/go1.23.3.linux-amd64.tar.gz got %s", url)
	}
}
