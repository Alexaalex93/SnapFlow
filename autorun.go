package main

import (
	"errors"
	"os"

	"golang.org/x/sys/windows/registry"
)

const regRunKey = `SOFTWARE\Microsoft\Windows\CurrentVersion\Run`

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
	return v == os.Args[0], nil
}

func AutoRunEnable() error {
	rk, err := registry.OpenKey(registry.CURRENT_USER, regRunKey, registry.WRITE)
	if err != nil {
		return err
	}
	defer rk.Close()
	return rk.SetStringValue(appName, os.Args[0])
}

func AutoRunDisable() error {
	rk, err := registry.OpenKey(registry.CURRENT_USER, regRunKey, registry.WRITE)
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
