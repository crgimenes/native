//go:build unix

// Unix backend (macOS, Linux, *BSD): an flock'd lock file is the lock — flock is
// released automatically when the process dies, so there is no stale lock to
// clean up — and a Unix-domain socket carries the argument hand-off. Pure
// standard library, no purego.

package singleinstance

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"syscall"
)

// runtimeDir picks a per-user directory for the lock and socket. XDG_RUNTIME_DIR
// is the right place on Linux; elsewhere the temp dir is the portable fallback.
func runtimeDir() string {
	if d := os.Getenv("XDG_RUNTIME_DIR"); d != "" {
		return d
	}
	return os.TempDir()
}

func paths(id string) (lock, sock string) {
	stem := filepath.Join(runtimeDir(), "native-si-"+keyFor(id))
	return stem + ".lock", stem + ".sock"
}

func acquire(id string, opts Options) (*Instance, error) {
	lockPath, sockPath := paths(id)
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, ErrAlreadyRunning
		}
		return nil, err
	}

	// We hold the lock: we are the primary. A previous primary that crashed may
	// have left a stale socket file; since we hold the lock, removing it is safe.
	_ = os.Remove(sockPath)
	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
		return nil, err
	}
	go serve(ln, opts.OnMessage)

	return &Instance{release: func() error {
		_ = ln.Close() // unblocks the Accept loop and unlinks the socket
		_ = os.Remove(sockPath)
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		err := f.Close()
		_ = os.Remove(lockPath)
		return err
	}}, nil
}

func serve(ln net.Listener, onMessage func([]string)) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return // listener closed on Release
		}
		go func() {
			defer func() { _ = conn.Close() }()
			data, err := io.ReadAll(conn)
			if err != nil {
				return
			}
			var args []string
			if json.Unmarshal(data, &args) == nil && onMessage != nil {
				onMessage(args)
			}
		}()
	}
}

func send(id string, args []string) error {
	_, sockPath := paths(id)
	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		return err // no instance listening (dial refused / socket missing)
	}
	defer func() { _ = conn.Close() }()
	data, err := json.Marshal(args)
	if err != nil {
		return err
	}
	_, err = conn.Write(data)
	return err
}
