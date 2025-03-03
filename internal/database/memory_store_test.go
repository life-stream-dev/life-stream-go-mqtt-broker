package database

import "testing"

func TestMemorySessionStore(t *testing.T) {
	store := NewMemoryStore()
	store2 := NewMemoryStore()
	if store != store2 {
		t.Fatal("memory session store does not match")
	}

	store.SaveSession(NewSessionData("1"))
	store.SaveSession(NewSessionData("2"))
	store.SaveSession(NewSessionData("3"))

	_, err := store.GetSession("2")
	if err != nil {
		t.Fatal("Except got client id 2, but got error")
	}

	store.DeleteSession("1")
	_, err = store.GetSession("1")
	if err == nil {
		t.Fatal("Except not fount error, but got nil")
	}
}
