package profile

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const AuthFileName = "auth.json"

type AuthSummary struct {
	Present   bool
	AccountID string
	Email     string
	Name      string
}

func AuthPath(codexHome string) string {
	return filepath.Join(codexHome, AuthFileName)
}

func EnsureHome(codexHome string) error {
	return os.MkdirAll(codexHome, 0o755)
}

func HasAuth(codexHome string) bool {
	_, err := os.Stat(AuthPath(codexHome))
	return err == nil
}

// ReadAuthSummary reads the auth.json file from codexHome and returns account
// metadata. It may return BOTH a partially populated AuthSummary (with at least
// Present=true) AND a non-nil error when the auth file exists but cannot be
// fully decoded. Callers should check summary.Present first and treat the error
// as a degraded-info signal rather than a hard failure.
func ReadAuthSummary(codexHome string) (AuthSummary, error) {
	path := AuthPath(codexHome)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return AuthSummary{}, nil
		}
		return AuthSummary{}, err
	}

	summary := AuthSummary{Present: true}
	var authFile struct {
		Tokens struct {
			AccountID string `json:"account_id"`
			IDToken   string `json:"id_token"`
		} `json:"tokens"`
	}
	if err := json.Unmarshal(data, &authFile); err != nil {
		return summary, err
	}

	summary.AccountID = authFile.Tokens.AccountID

	if authFile.Tokens.IDToken == "" {
		return summary, nil
	}

	payload, err := decodeJWTPayload(authFile.Tokens.IDToken)
	if err != nil {
		return summary, err
	}
	summary.Email = payload.Email
	summary.Name = payload.Name
	return summary, nil
}

func DefaultImportSource() (string, error) {
	if codexHome := os.Getenv("CODEX_HOME"); codexHome != "" {
		return filepath.Join(codexHome, AuthFileName), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".codex", AuthFileName), nil
}

func ResolveImportSource(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		path = filepath.Join(path, AuthFileName)
		info, err = os.Stat(path)
		if err != nil {
			return "", err
		}
		if info.IsDir() {
			return "", fmt.Errorf("%s is a directory; expected auth.json", path)
		}
	}
	return path, nil
}

func ImportAuth(codexHome string, sourcePath string) (string, string, error) {
	resolvedSource, err := ResolveImportSource(sourcePath)
	if err != nil {
		return "", "", err
	}
	if err := EnsureHome(codexHome); err != nil {
		return "", "", err
	}

	destPath := AuthPath(codexHome)
	if err := copyFile(resolvedSource, destPath, 0o600); err != nil {
		return "", "", err
	}
	return resolvedSource, destPath, nil
}

func copyFile(src, dest string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	tmp := dest + ".tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dest)
}

func decodeJWTPayload(token string) (struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}, error) {
	var payload struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}

	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return payload, fmt.Errorf("invalid JWT format")
	}

	decoded, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return payload, err
	}
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return payload, err
	}
	return payload, nil
}
