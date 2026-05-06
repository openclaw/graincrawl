package granola

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"regexp"
)

type AppInfo struct {
	Path      string `json:"path"`
	Installed bool   `json:"installed"`
	Version   string `json:"version,omitempty"`
	BundleID  string `json:"bundle_id,omitempty"`
}

func InspectApp(appPath string) AppInfo {
	if appPath == "" {
		appPath = DefaultAppPath
	}
	info := AppInfo{Path: appPath}
	if _, err := os.Stat(appPath); err == nil {
		info.Installed = true
	}
	plist := filepath.Join(appPath, "Contents", "Info.plist")
	if b, err := os.ReadFile(plist); err == nil {
		info.Version = plistString(b, "CFBundleShortVersionString")
		info.BundleID = plistString(b, "CFBundleIdentifier")
	}
	return info
}

func plistString(b []byte, key string) string {
	var doc struct {
		Dict struct {
			Inner string `xml:",innerxml"`
		} `xml:"dict"`
	}
	if err := xml.Unmarshal(b, &doc); err != nil {
		return ""
	}
	re := regexp.MustCompile(`<key>` + regexp.QuoteMeta(key) + `</key>\s*<string>([^<]+)</string>`)
	match := re.FindStringSubmatch(doc.Dict.Inner)
	if len(match) != 2 {
		return ""
	}
	return match[1]
}
