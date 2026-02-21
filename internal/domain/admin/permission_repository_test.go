package admin

import (
	"reflect"
	"testing"
)

func TestSplitFilterCSV(t *testing.T) {
	raw := " admin.account.read,admin.account.read, ,admin.stats.read "
	got := splitFilterCSV(raw)
	want := []string{"admin.account.read", "admin.stats.read"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected split result: got=%v want=%v", got, want)
	}
}
