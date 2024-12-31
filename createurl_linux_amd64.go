//go:build linux && amd64

package main

import "fmt"

func createUrl(version string) string {
	return fmt.Sprintf("https://go.dev/dl/go%v.linux-amd64.tar.gz", version)
}
