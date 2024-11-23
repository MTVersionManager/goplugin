package goplugin

import (
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
