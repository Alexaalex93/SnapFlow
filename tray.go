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
	_ "embed"
	"fmt"

	"github.com/getlantern/systray"
	"github.com/gonutz/w32/v2"
)

//go:embed assets/tray_icon.ico
var icon []byte

func initTray() {
	systray.Register(onReady, onExit)
}

func onReady() {
	systray.SetIcon(icon)
	systray.SetTitle(ProductName)
	systray.SetTooltip(ProductName)

	autorun, err := AutoRunEnabled()
	if err != nil {
		panic(err)
	}

	licenseLabel := "License: Free"
	if entitlements != nil && entitlements.IsPro() {
		licenseLabel = "License: Pro"
	}
	mLicense := systray.AddMenuItem(licenseLabel, "")
	mLicense.Disable()

	mRepo := systray.AddMenuItem("Documentation", "")
	go func() {
		for range mRepo.ClickedCh {
			if err := w32.ShellExecute(0, "open", ProductDocsURL, "", "", w32.SW_SHOWNORMAL); err != nil {
				fmt.Printf("failed to launch browser: (%d), %v\n", w32.GetLastError(), err)
			}
		}
	}()

	mUpgrade := systray.AddMenuItem("Upgrade to Pro", "")
	go func() {
		for range mUpgrade.ClickedCh {
			if err := w32.ShellExecute(0, "open", ProductUpgradeURL, "", "", w32.SW_SHOWNORMAL); err != nil {
				fmt.Printf("failed to launch upgrade url: (%d), %v\n", w32.GetLastError(), err)
			}
			if appConfig != nil {
				appConfig.UpgradeEntrySeen = true
				if appConfigStore != nil {
					_ = appConfigStore.Save(appConfig)
				}
			}
		}
	}()

	mOpenConfig := systray.AddMenuItem("Open Config", "")
	go func() {
		for range mOpenConfig.ClickedCh {
			if appConfigStore == nil {
				continue
			}
			if err := w32.ShellExecute(0, "open", appConfigStore.path, "", "", w32.SW_SHOWNORMAL); err != nil {
				fmt.Printf("failed to launch config file: (%d), %v\n", w32.GetLastError(), err)
			}
		}
	}()

	systray.AddSeparator()

	mAutoRun := systray.AddMenuItemCheckbox("Run on startup", "", autorun)
	go func() {
		for range mAutoRun.ClickedCh {
			if mAutoRun.Checked() {
				if err := AutoRunDisable(); err != nil {
					mAutoRun.SetTitle(err.Error())
					fmt.Printf("warn: autorun disable: %v\n", err)
					continue
				}
				fmt.Println("disabled autorun")
				mAutoRun.Uncheck()
			} else {
				if err := AutoRunEnable(); err != nil {
					mAutoRun.SetTitle(err.Error())
					fmt.Printf("warn: autorun enable: %v\n", err)
					continue
				}
				fmt.Println("enabled autorun")
				mAutoRun.Check()
			}

		}
	}()

	systray.AddSeparator()

	mQuit := systray.AddMenuItem("Quit", "")
	go func() {
		<-mQuit.ClickedCh
		fmt.Println("clicked Quit")
		systray.Quit()
	}()

	fmt.Println("tray ready")
}

func onExit() {
	if dragManager != nil {
		dragManager.Stop()
		if dragManager.overlay != nil {
			dragManager.overlay.Close()
		}
	}
	fmt.Println("onExit invoked")
}
