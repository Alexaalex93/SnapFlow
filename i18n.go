package main

import "syscall"

// Native-UI strings (tray menu, notifications). The settings page has its
// own JS dictionary; both follow AppConfig.Language, or the Windows UI
// language when that is empty.

var nativeStrings = map[string]map[string]string{
	"en": {
		"menuSettings":    "Settings…",
		"menuSnapDrag":    "Snap windows by dragging",
		"menuWinSnap":     "Windows Snap (conflicts with SnapFlow)",
		"menuAutoRun":     "Launch at login",
		"menuOpenConfig":  "Open config file",
		"menuUpdates":     "Check for updates…",
		"menuQuit":        "Quit SnapFlow",
		"tip":             "SnapFlow — drag windows to the screen edges",
		"tipPaused":       "SnapFlow — drag snapping paused",
		"cfgReset":        "Your config.json had an error, so default settings were loaded. The old file was saved as config.json.bad.",
		"conflictTitle":   "Some shortcuts are taken",
		"conflictBody":    "%d shortcut(s) are used by another app: %s. Click here to change them in Settings.",
		"conflictArrows":  " Ctrl+Alt+Arrows are often reserved by the graphics driver (screen rotation).",
		"elevatedTitle":   "Can't move this window",
		"elevatedBody":    "Windows of apps running as administrator can't be moved by SnapFlow unless SnapFlow also runs as administrator.",
		"updateTitle":     "SnapFlow %s is available",
		"updateBody":      "You have %s. Click here to open the download page.",
		"upToDateTitle":   "SnapFlow is up to date",
		"upToDateBody":    "You have the latest version (%s).",
		"updateErrTitle":  "Couldn't check for updates",
		"winSnapApplied":  "Windows Snap is now %s.",
		"winSnapEnabled":  "enabled",
		"winSnapDisabled": "disabled",
	},
	"es": {
		"menuSettings":    "Ajustes…",
		"menuSnapDrag":    "Ajustar ventanas arrastrando",
		"menuWinSnap":     "Windows Snap (choca con SnapFlow)",
		"menuAutoRun":     "Iniciar al iniciar sesión",
		"menuOpenConfig":  "Abrir archivo de configuración",
		"menuUpdates":     "Buscar actualizaciones…",
		"menuQuit":        "Salir de SnapFlow",
		"tip":             "SnapFlow — arrastra ventanas a los bordes",
		"tipPaused":       "SnapFlow — ajuste por arrastre en pausa",
		"cfgReset":        "Tu config.json tenía un error y se cargaron los valores por defecto. El archivo anterior se guardó como config.json.bad.",
		"conflictTitle":   "Algunos atajos están ocupados",
		"conflictBody":    "%d atajo(s) los usa otra aplicación: %s. Haz clic aquí para cambiarlos en Ajustes.",
		"conflictArrows":  " Ctrl+Alt+Flechas suele estar reservado por el driver gráfico (rotación de pantalla).",
		"elevatedTitle":   "No se puede mover esta ventana",
		"elevatedBody":    "SnapFlow no puede mover ventanas de apps ejecutadas como administrador, salvo que SnapFlow también se ejecute como administrador.",
		"updateTitle":     "SnapFlow %s disponible",
		"updateBody":      "Tienes la %s. Haz clic aquí para abrir la página de descarga.",
		"upToDateTitle":   "SnapFlow está actualizado",
		"upToDateBody":    "Tienes la última versión (%s).",
		"updateErrTitle":  "No se pudo comprobar si hay actualizaciones",
		"winSnapApplied":  "Windows Snap ahora está %s.",
		"winSnapEnabled":  "activado",
		"winSnapDisabled": "desactivado",
	},
}

var procGetUserDefaultUILanguage = syscall.NewLazyDLL("kernel32.dll").NewProc("GetUserDefaultUILanguage")

// systemLang returns "es" when Windows' UI language is Spanish, else "en".
func systemLang() string {
	r, _, _ := procGetUserDefaultUILanguage.Call()
	if r&0x3FF == 0x0A { // LANG_SPANISH primary language id
		return "es"
	}
	return "en"
}

// uiLang is the effective language: configured, or Windows' own.
func uiLang() string {
	if c := liveCfg.Load(); c != nil && (c.Language == "en" || c.Language == "es") {
		return c.Language
	}
	return systemLang()
}

func tr(key string) string {
	if s, ok := nativeStrings[uiLang()][key]; ok {
		return s
	}
	return nativeStrings["en"][key]
}
