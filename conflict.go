package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// notifyShortcutConflicts tells the user (via a non-blocking notification,
// not a modal dialog) which shortcuts another app already owns.
//
// On startup the same set of conflicts is only reported once — the
// signature is remembered in the config — so SnapFlow doesn't nag on every
// login. explicit=true (shortcuts just changed in Settings) always reports.
// Must run on the main thread.
func notifyShortcutConflicts(failed []HotKey, explicit bool) {
	if len(failed) == 0 {
		return
	}
	names := make([]string, 0, len(failed))
	hasArrow := false
	for _, hk := range failed {
		names = append(names, hk.String())
		if hk.vk >= 0x25 && hk.vk <= 0x28 && hk.mod&(MOD_CONTROL|MOD_ALT) == MOD_CONTROL|MOD_ALT {
			hasArrow = true
		}
	}
	sort.Strings(names)
	sig := strings.Join(names, "|")
	if !explicit && appCfg.NotifiedConflicts == sig {
		return
	}
	if appCfg.NotifiedConflicts != sig {
		c := appCfg.clone()
		c.NotifiedConflicts = sig
		appCfg = c
		liveCfg.Store(c)
		if appCfgStore != nil {
			_ = appCfgStore.Save(c)
		}
	}

	list := strings.Join(names, ", ")
	if len(names) > 4 {
		list = strings.Join(names[:4], ", ") + "…"
	}
	body := fmt.Sprintf(tr("conflictBody"), len(failed), list)
	if hasArrow {
		body += tr("conflictArrows")
	}
	trayNotify(tr("conflictTitle"), body, openSettings)
}

var elevatedOnce sync.Once

// notifyElevatedOnce explains (once per session) why a window of an
// elevated app could not be moved — Windows blocks that (UIPI).
func notifyElevatedOnce() {
	elevatedOnce.Do(func() {
		trayNotify(tr("elevatedTitle"), tr("elevatedBody"))
	})
}
