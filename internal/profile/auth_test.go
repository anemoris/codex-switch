package profile

import (
	"os"
	"path/filepath"
	"testing"

	"codex-switch/internal/testutil"
)

func TestImportAuthFromDirectory(t *testing.T) {
	dir := t.TempDir()
	sourceDir := filepath.Join(dir, "source")
	destDir := filepath.Join(dir, "dest")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, AuthFileName), []byte("token"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, destPath, err := ImportAuth(destDir, sourceDir)
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "token" {
		t.Fatalf("unexpected auth contents: %q", data)
	}
}

func TestReadAuthSummaryParsesAccountMetadata(t *testing.T) {
	dir := t.TempDir()
	authPath := filepath.Join(dir, AuthFileName)
	jwt := testutil.TestJWT(`{"email":"person@example.com","name":"Person"}`)
	data := `{"tokens":{"account_id":"acct_123","id_token":"` + jwt + `"}}`
	if err := os.WriteFile(authPath, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	summary, err := ReadAuthSummary(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !summary.Present {
		t.Fatal("expected auth to be present")
	}
	if summary.AccountID != "acct_123" {
		t.Fatalf("unexpected account ID: %q", summary.AccountID)
	}
	if summary.Email != "person@example.com" {
		t.Fatalf("unexpected email: %q", summary.Email)
	}
	if summary.Name != "Person" {
		t.Fatalf("unexpected name: %q", summary.Name)
	}
}

func TestReadAuthSummaryReturnsZeroValueWhenMissing(t *testing.T) {
	summary, err := ReadAuthSummary(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if summary.Present {
		t.Fatalf("expected missing auth summary, got %#v", summary)
	}
}

func TestReadAuthSummaryReturnsPresentWhenUnreadable(t *testing.T) {
	dir := t.TempDir()
	authPath := filepath.Join(dir, AuthFileName)
	if err := os.WriteFile(authPath, []byte("{invalid"), 0o600); err != nil {
		t.Fatal(err)
	}

	summary, err := ReadAuthSummary(dir)
	if err == nil {
		t.Fatal("expected decode error for unreadable auth file")
	}
	if !summary.Present {
		t.Fatalf("expected unreadable auth file to still be marked present, got %#v", summary)
	}
	if summary.AccountID != "" || summary.Email != "" || summary.Name != "" {
		t.Fatalf("expected unreadable auth summary to be partial, got %#v", summary)
	}
}
