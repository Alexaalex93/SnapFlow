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

func openReleasesPage() {
	w32.ShellExecute(0, "open", releasesURL, "", "", w32.SW_SHOWNORMAL)
}

// checkForUpdates queries GitHub for the latest release and reports the
// result as a tray notification (never a blocking dialog). If silent=true
// (startup check) it only speaks up when an update is available.
func checkForUpdates(silent bool) {
	latest, err := fetchLatestVersion()
	if err != nil {
		if !silent {
			trayNotify(tr("updateErrTitle"), err.Error(), openReleasesPage)
		}
		return
	}

	current := strings.TrimPrefix(appVersion, "v")
	latestClean := strings.TrimPrefix(latest, "v")

	if !isNewerVersion(latestClean, current) {
		if !silent {
			trayNotify(tr("upToDateTitle"), fmt.Sprintf(tr("upToDateBody"), current))
		}
		return
	}
	trayNotify(fmt.Sprintf(tr("updateTitle"), latestClean),
		fmt.Sprintf(tr("updateBody"), current), openReleasesPage)
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
		return "", fmt.Errorf("no releases found")
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
// A "dev" build is never offered updates.
func isNewerVersion(candidate, current string) bool {
	if current == "dev" || current == "" {
		return false
	}
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
