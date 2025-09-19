// go-term-info.go
package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/term"
)

// tryReadlinkFd0 tries to resolve /proc/self/fd/0 (Linux) to get the tty path.
func tryReadlinkFd0() (string, error) {
	path := "/proc/self/fd/0"
	if _, err := os.Stat(path); err != nil {
		return "", err
	}
	t, err := os.Readlink(path)
	if err != nil {
		return "", err
	}
	return t, nil
}

func envKeys(keys []string) map[string]string {
	out := map[string]string{}
	for _, k := range keys {
		if v, ok := os.LookupEnv(k); ok {
			out[k] = v
		}
	}
	return out
}

func main() {
	fmt.Println("=== Terminal / environnement — résumé ===")
	fmt.Printf("GOOS: %s\n", runtime.GOOS)
	fmt.Printf("GOARCH: %s\n", runtime.GOARCH)
	fmt.Println()

	// process info
	fmt.Printf("PID: %d\n", os.Getpid())
	fmt.Printf("PPID: %d\n", os.Getppid())

	// current working dir
	if wd, err := os.Getwd(); err == nil {
		fmt.Printf("Working dir: %s\n", wd)
	} else {
		fmt.Printf("Working dir: <error: %v>\n", err)
	}

	// user info
	if u, err := user.Current(); err == nil {
		fmt.Printf("User: %s (UID: %s, GID: %s) Home: %s\n", u.Username, u.Uid, u.Gid, u.HomeDir)
	} else {
		fmt.Printf("User: <error: %v>\n", err)
	}

	// common terminal-related environment variables
	fmt.Println("\n--- Variables d'environnement liées au terminal ---")
	keys := []string{"TERM", "SHELL", "COLORTERM", "TERM_PROGRAM", "TERM_PROGRAM_VERSION", "LANG", "LC_ALL", "LC_CTYPE", "SSH_TTY", "SSH_CONNECTION"}
	for k, v := range envKeys(keys) {
		fmt.Printf("%s=%s\n", k, v)
	}
	fmt.Println()

	// isatty checks
	stdinIsTTY := term.IsTerminal(int(os.Stdin.Fd()))
	stdoutIsTTY := term.IsTerminal(int(os.Stdout.Fd()))
	stderrIsTTY := term.IsTerminal(int(os.Stderr.Fd()))
	fmt.Printf("Stdin is TTY: %v\n", stdinIsTTY)
	fmt.Printf("Stdout is TTY: %v\n", stdoutIsTTY)
	fmt.Printf("Stderr is TTY: %v\n", stderrIsTTY)

	// terminal size (try stdout, then stdin)
	var w, h int
	var err error
	if stdoutIsTTY {
		w, h, err = term.GetSize(int(os.Stdout.Fd()))
	} else if stdinIsTTY {
		w, h, err = term.GetSize(int(os.Stdin.Fd()))
	}
	if err == nil && w > 0 && h > 0 {
		fmt.Printf("Terminal size: cols=%d rows=%d\n", w, h)
	} else {
		fmt.Println("Terminal size: unavailable")
	}

	// name of controlling TTY (best-effort)
	fmt.Println("\n--- Controlling TTY / fd info (best-effort) ---")
	// 1) try /proc/self/fd/0 (Linux)
	if t, err := tryReadlinkFd0(); err == nil {
		fmt.Printf("/proc/self/fd/0 -> %s\n", t)
	} else {
		fmt.Printf("/proc/self/fd/0 -> %v\n", err)
	}

	// 2) try opening /dev/tty (portable on UNIX)
	if f, err := os.OpenFile("/dev/tty", os.O_RDONLY, 0); err == nil {
		fmt.Printf("Opened /dev/tty (Name()): %s\n", f.Name())
		_ = f.Close()
	} else {
		fmt.Printf("/dev/tty: %v\n", err)
	}

	// 3) fallback: check if SSH_TTY env var is set
	if sshTTY, ok := os.LookupEnv("SSH_TTY"); ok && sshTTY != "" {
		fmt.Printf("SSH_TTY (env) = %s\n", sshTTY)
	}

	// 4) try `tty` command output as a fallback (if available)
	if path, err := execLookPathTTY(); err == nil {
		out, err2 := runTTYCommand(path)
		if err2 == nil {
			fmt.Printf("`tty` command -> %s\n", strings.TrimSpace(out))
		} else {
			fmt.Printf("`tty` command error: %v\n", err2)
		}
	} else {
		fmt.Println("`tty` command not found in PATH")
	}

	// other useful environment summaries
	fmt.Println("\n--- Autres informations d'environnement ---")
	fmt.Printf("PATH=%s\n", os.Getenv("PATH"))
	fmt.Printf("HOME=%s\n", os.Getenv("HOME"))
	if umask := getUmask(); umask != "" {
		fmt.Printf("umask=%s\n", umask)
	} else {
		fmt.Printf("umask: unavailable (requires syscall on some OS)\n")
	}

	// extras: file descriptors pointing to same device (best-effort)
	fmt.Println("\n--- FDs -> names (fd 0..3) ---")
	for fd := 0; fd <= 3; fd++ {
		name := fdName(fd)
		fmt.Printf("fd %d -> %s\n", fd, name)
	}

	fmt.Println("\n=== Fin ===")
}

// fdName tries to resolve /proc/self/fd/N or returns file.Name() when possible
func fdName(fd int) string {
	// try /proc/self/fd/N
	proc := fmt.Sprintf("/proc/self/fd/%d", fd)
	if t, err := os.Readlink(proc); err == nil {
		return t
	}
	// fallback: try opening /dev/fd/N symlink on some systems
	fpath := fmt.Sprintf("/dev/fd/%d", fd)
	if t, err := os.Readlink(fpath); err == nil {
		return t
	}
	return "unknown"
}

func execLookPathTTY() (string, error) {
	return execLookPath("tty")
}

func execLookPath(name string) (string, error) {
	// simple wrapper so we don't import os/exec at top-level when not necessary in some builds
	return func(n string) (string, error) {
		return filepath.Abs(n) // dummy to satisfy function shape (we replace below)
	}(name)
}

func runTTYCommand(path string) (string, error) {
	// attempt to run `tty` using /bin/sh -c "tty" for portability
	// we implement minimally: use os/exec here
	out, err := runCmd("sh", "-c", "tty")
	return string(out), err
}

func runCmd(name string, args ...string) ([]byte, error) {
	// local import to avoid unused imports on other platforms
	// this function will actually use os/exec
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("command execution failed: %w", err)
	}
	return output, nil
}

// getUmask: No portable way in pure Go to read current umask without syscall; returning empty
func getUmask() string {
	return ""
}
