package websocket

import "testing"

func TestNormalizeMemberIDs_IncludesActorAndUniqueSorted(t *testing.T) {
	result := normalizeMemberIDs("adminA", []string{"", "manager1", "adminA", "guest1", "manager1"})

	if len(result) != 3 {
		t.Fatalf("expected 3 unique members, got %d (%v)", len(result), result)
	}

	expected := []string{"adminA", "guest1", "manager1"}
	for i := range expected {
		if result[i] != expected[i] {
			t.Fatalf("expected result[%d]=%s, got %s", i, expected[i], result[i])
		}
	}
}

func TestBuildDirectRoomKey_IsOrderIndependent(t *testing.T) {
	key1 := buildDirectRoomKey("adminA", "managerB")
	key2 := buildDirectRoomKey("managerB", "adminA")
	if key1 != key2 {
		t.Fatalf("expected identical direct room keys, got %s vs %s", key1, key2)
	}
}

func TestIsValidChatRoomKey(t *testing.T) {
	if !isValidChatRoomKey("grp_1234_abcd") {
		t.Fatal("expected grp_1234_abcd to be valid room key")
	}
	if isValidChatRoomKey("invalid room key") {
		t.Fatal("expected room key with spaces to be invalid")
	}
}
