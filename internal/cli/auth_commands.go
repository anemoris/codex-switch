package cli

import (
	"crypto/sha256"
	"errors"
	"flag"
	"fmt"
	"os"

	"codex-switch/internal/config"
	"codex-switch/internal/profile"
)

func (a *App) runExec(args []string) error {
	return a.execProfileCommand("run", args, []string{"codex"})
}

func (a *App) runLogin(args []string) error {
	var copyFromCurrent bool
	profileName, commandArgs, err := a.parseProfileCommandArgsWithFlags("login", args, []string{"codex", "login"}, func(fs *flag.FlagSet) {
		fs.BoolVar(&copyFromCurrent, "copy-from-current", false, "copy auth.json from the current CODEX_HOME instead of running codex login")
	})
	if err != nil {
		return err
	}

	p, resolvedName, err := a.lookupProfileOrDefault(profileName)
	if err != nil {
		return err
	}
	if err := profile.EnsureHome(p.CodexHome); err != nil {
		return err
	}

	if copyFromCurrent {
		if len(commandArgs) != 0 && !(len(commandArgs) == 2 && commandArgs[0] == "codex" && commandArgs[1] == "login") {
			return errors.New("usage: codex-switch login [<profile>] [--copy-from-current]")
		}
		source, err := profile.DefaultImportSource()
		if err != nil {
			return err
		}
		resolvedSource, destPath, err := profile.ImportAuth(p.CodexHome, source)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(a.stdout, "Copied current auth into profile %s\n", resolvedName); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(a.stdout, "source=%s\n", resolvedSource); err != nil {
			return err
		}
		_, err = fmt.Fprintf(a.stdout, "dest=%s\n", destPath)
		return err
	}

	authPath := profile.AuthPath(p.CodexHome)
	before, err := readAuthFileState(authPath)
	if err != nil {
		return err
	}
	if err := a.runCommandInProfile(p, commandArgs); err != nil {
		return err
	}
	after, err := readAuthFileState(authPath)
	if err != nil {
		return err
	}
	if !after.Exists {
		return fmt.Errorf("login finished but %s was not created", authPath)
	}
	if before.Exists && before.Hash == after.Hash {
		return fmt.Errorf("login finished but %s was not updated", authPath)
	}
	_, err = fmt.Fprintf(a.stdout, "Login completed for profile %s\n", resolvedName)
	return err
}

func (a *App) runImportAuth(args []string) error {
	name, rest := consumeOptionalName(args)

	fs := a.newFlagSet("import-auth")
	var source string
	fs.StringVar(&source, "from", "", "source auth.json path or directory containing auth.json")

	if err := fs.Parse(rest); err != nil {
		return err
	}
	if name == "" {
		if fs.NArg() != 1 {
			return usageError("usage: codex-switch import-auth <profile> [--from path]")
		}
		name = fs.Arg(0)
	} else if fs.NArg() != 0 {
		return usageError("usage: codex-switch import-auth <profile> [--from path]")
	}

	p, err := a.lookupProfile(name)
	if err != nil {
		return err
	}

	if source == "" {
		source, err = profile.DefaultImportSource()
		if err != nil {
			return err
		}
	} else {
		source, err = config.ExpandPath(source)
		if err != nil {
			return err
		}
	}

	resolvedSource, destPath, err := profile.ImportAuth(p.CodexHome, source)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(a.stdout, "Imported auth for profile %s\n", name); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(a.stdout, "source=%s\n", resolvedSource); err != nil {
		return err
	}
	_, err = fmt.Fprintf(a.stdout, "dest=%s\n", destPath)
	return err
}

type authFileState struct {
	Exists bool
	Hash   [32]byte
}

func readAuthFileState(path string) (authFileState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return authFileState{}, nil
		}
		return authFileState{}, err
	}
	return authFileState{
		Exists: true,
		Hash:   sha256.Sum256(data),
	}, nil
}
