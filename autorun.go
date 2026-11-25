package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const regRunKey = `SOFTWARE\Microsoft\Windows\CurrentVersion\Run`

// exePath is the absolute path of the running executable. (os.Args[0] —
// used before — can be relative or differ in case/quoting from what the
// installer writes, which made the "launch at login" checkbox show the
// wrong state.)
func exePath() string {
	p, err := os.Executable()
	if err != nil {
		return os.Args[0]
	}
	if r, err := filepath.EvalSymlinks(p); err == nil {
		p = r
	}
	return p
}

func normPath(p string) string {
	p = strings.TrimSpace(p)
	p = strings.Trim(p, `"`)
	return strings.ToLower(filepath.Clean(p))
}

func AutoRunEnabled() (bool, error) {
	rk, err := registry.OpenKey(registry.CURRENT_USER, regRunKey, registry.QUERY_VALUE)
	if err != nil {
		return false, err
	}
	defer rk.Close()
	v, _, err := rk.GetStringValue(appName)
	if errors.Is(err, registry.ErrNotExist) || errors.Is(err, registry.ErrUnexpectedType) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return normPath(v) == normPath(exePath()), nil
}

func AutoRunEnable() error {
	rk, err := registry.OpenKey(registry.CURRENT_USER, regRunKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer rk.Close()
	// Quoted, so paths with spaces work.
	return rk.SetStringValue(appName, `"`+exePath()+`"`)
}

func AutoRunDisable() error {
	rk, err := registry.OpenKey(registry.CURRENT_USER, regRunKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer rk.Close()
	err = rk.DeleteValue(appName)
	if errors.Is(err, registry.ErrNotExist) {
		return nil
	}
	return err
}
