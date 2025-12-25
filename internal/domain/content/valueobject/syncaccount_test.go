package valueobject

import (
	"testing"
)

func TestSyncAccountSetHash(t *testing.T) {
	account := &SyncAccount{
		License: "MDF-ABCD-EFGH-JKLM",
	}

	account.SetHash()
	if account.Hash == "" {
		t.Error("SetHash() should set a non-empty hash")
	}
	if len(account.Hash) != 32 {
		t.Errorf("Hash length = %v, want 32", len(account.Hash))
	}
}

func TestSyncAccountSetSlug(t *testing.T) {
	account := &SyncAccount{
		License: "MDF-ABCD-EFGH-JKLM",
	}

	account.SetSlug("")
	expected := "MDF-ABCD-EFGH-JKLM"
	if account.Slug != expected {
		t.Errorf("SetSlug() = %v, want %v", account.Slug, expected)
	}
}

func TestSyncAccountString(t *testing.T) {
	account := &SyncAccount{
		Email:  "abcd-efgh-jklm@mdfriday.com",
		DBName: "userdb-abcd1234",
	}

	expected := "abcd-efgh-jklm@mdfriday.com - userdb-abcd1234"
	if got := account.String(); got != expected {
		t.Errorf("String() = %v, want %v", got, expected)
	}
}

func TestSyncAccountIndexContent(t *testing.T) {
	account := &SyncAccount{}
	if !account.IndexContent() {
		t.Error("IndexContent() should return true")
	}
}

