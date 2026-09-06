package renderer

import "testing"

func hasMenuEntry(menus []map[string]interface{}, href, label string) bool {
	for _, group := range menus {
		rawItems, ok := group["Items"]
		if !ok {
			continue
		}

		items, ok := rawItems.([]map[string]string)
		if !ok {
			return false
		}

		for _, item := range items {
			if item["Href"] == href && item["Label"] == label {
				return true
			}
		}
	}

	return false
}

func menuEntryIndex(menus []map[string]interface{}, href, label string) int {
	for _, group := range menus {
		rawItems, ok := group["Items"]
		if !ok {
			continue
		}

		items, ok := rawItems.([]map[string]string)
		if !ok {
			return -1
		}

		for i, item := range items {
			if item["Href"] == href && item["Label"] == label {
				return i
			}
		}
	}

	return -1
}

func TestMakeUnLoginMenu_RemovesHomeEntry(t *testing.T) {
	menus := MakeUnLoginMenu("localhost:8080")
	if hasMenuEntry(menus, "/", "홈") {
		t.Fatalf("expected guest menu to omit redundant home entry")
	}
}
