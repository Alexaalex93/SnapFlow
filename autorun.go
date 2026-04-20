// Copyright 2022 Ahmet Alp Balkan
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"errors"
	"os"

	"golang.org/x/sys/windows/registry"
)

const (
	AutoRunName    = ProductName
	legacyAutoRun  = "RectangleWin"
	regKey         = `SOFTWARE\Microsoft\Windows\CurrentVersion\Run`
)

func self() string {
	return os.Args[0]
}

func AutoRunEnabled() (bool, error) {
	rk, err := registry.OpenKey(registry.CURRENT_USER, regKey, registry.QUERY_VALUE)
	if err != nil {
		return false, err
	}
	defer rk.Close()

	for _, name := range []string{AutoRunName, legacyAutoRun} {
		v, _, err := rk.GetStringValue(name)
		if errors.Is(err, registry.ErrNotExist) || errors.Is(err, registry.ErrUnexpectedType) {
			continue
		}
		if err != nil {
			return false, err
		}
		if v == self() {
			return true, nil
		}
	}
	return false, nil
}

func AutoRunDisable() error {
	rk, err := registry.OpenKey(registry.CURRENT_USER, regKey, registry.WRITE)
	if err != nil {
		return err
	}
	defer rk.Close()

	for _, name := range []string{AutoRunName, legacyAutoRun} {
		err = rk.DeleteValue(name)
		if errors.Is(err, registry.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func AutoRunEnable() error {
	rk, err := registry.OpenKey(registry.CURRENT_USER, regKey, registry.WRITE)
	if err != nil {
		return err
	}
	defer rk.Close()
	if err := rk.DeleteValue(legacyAutoRun); err != nil && !errors.Is(err, registry.ErrNotExist) {
		return err
	}
	return rk.SetStringValue(AutoRunName, self())
}
