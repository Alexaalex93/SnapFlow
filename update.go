package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gonutz/w32/v2"
)

const (
	releasesAPIURL = "https://api.github.com/repos/Alexaalex93/SnapFlow/releases/latest"
	releasesURL    = "https://github.com/Alexaalex93/SnapFlow/releases"
)

// checkForUpdates queries GitHub for the latest release and shows a result dialog.
// If silent=true, only shows a dialog when an update is available (used on startup).
func checkForUpdates(silent bool) {
	latest, err := fetchLatestVersion()
	if err != nil {
		if !silent {
			w32.MessageBox(0,
				"Could not check for updates:\n"+err.Error()+
					"\n\nYou can check manually at:\n"+releasesURL,
				appName+" — Update Check",
				w32.MB_ICONWARNING|w32.MB_OK)
		}
		return
	}

	current := strings.TrimPrefix(appVersion, "v")
	latestClean := strings.TrimPrefix(latest, "v")

	if !isNewerVersion(latestClean, current) {
		if !silent {
			w32.MessageBox(0,
				fmt.Sprintf("SnapFlow %s is up to date.", current),
				appName+" — Up to date",
				w32.MB_ICONINFORMATION|w32.MB_OK)
		}
		return
	}

	// New version available — show prompt regardless of silent flag.
	msg := fmt.Sprintf(
		"A new version of SnapFlow is available!\n\n"+
			"  Installed:  %s\n"+
			"  Available:  %s\n\n"+
			"Download and run the new installer to update.\n"+
			"SnapFlow will close automatically during the install.\n\n"+
			"Open download page now?",
		current, latestClean)

	result := w32.MessageBox(0, msg, appName+" — Update Available",
		w32.MB_ICONINFORMATION|w32.MB_YESNO)
	if result == w32.IDYES {
		w32.ShellExecute(0, "open", releasesURL, "", "", w32.SW_SHOWNORMAL)
	}
}

// fetchLatestVersion returns the tag_name of the latest GitHub release.
func fetchLatestVersion() (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", releasesAPIURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", appName+"/"+appVersion)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return "", fmt.Errorf("no releases found (repository may be private)")
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("invalid response: %w", err)
	}
	if release.TagName == "" {
		return "", fmt.Errorf("empty tag in response")
	}
	return release.TagName, nil
}

// isNewerVersion returns true if candidate is strictly newer than current.
// Both strings should be "MAJOR.MINOR.PATCH" without a leading "v".
func isNewerVersion(candidate, current string) bool {
	ca := parseSemVer(candidate)
	cu := parseSemVer(current)
	for i := 0; i < 3; i++ {
		if ca[i] > cu[i] {
			return true
		}
		if ca[i] < cu[i] {
			return false
		}
	}
	return false
}

func parseSemVer(v string) [3]int {
	// Strip pre-release / build-metadata suffixes (e.g. "1.2.3-beta" → "1.2.3").
	v = strings.SplitN(v, "-", 2)[0]
	v = strings.SplitN(v, "+", 2)[0]
	parts := strings.SplitN(v, ".", 3)
	var out [3]int
	for i, p := range parts {
		if i >= 3 {
			break
		}
		n, _ := strconv.Atoi(strings.TrimSpace(p))
		out[i] = n
	}
	return out
}
