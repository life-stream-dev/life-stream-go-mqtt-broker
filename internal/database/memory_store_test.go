package database

import "testing"

func TestMemorySessionStore(t *testing.T) {
	store := NewMemorySessionStore()
	store2 := NewMemorySessionStore()
	if store != store2 {
		t.Fatal("memory session store does not match")
	}

	store.Save(NewSessionData("1"))
	store.Save(NewSessionData("2"))
	store.Save(NewSessionData("3"))

	_, err := store.Get("2")
	if err != nil {
		t.Fatal("Except got client id 2, but got error")
	}

	store.Delete("1")
	_, err = store.Get("1")
	if err == nil {
		t.Fatal("Except not fount error, but got nil")
	}
}
