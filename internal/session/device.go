package session

import (
	"strings"
	"unicode/utf8"

	"github.com/mssola/useragent"
)

// Where the device info came from.
const (
	SourceClient    = "client"
	SourceUserAgent = "user_agent"
)

// Device describes the device of a session, e.g. {"name":"iPhone 15","os":"iOS 18","client":"MyApp 2.4"}.
type Device struct {
	Name   string `json:"name"`
	OS     string `json:"os"`
	Client string `json:"client"`
	Source string `json:"source"`
}

const maxDeviceFieldLength = 100

// DetectDevice uses the device info sent by the client. If the client sent nothing
// (e.g. curl or Postman), it is parsed from the User-Agent (docs/DECISION.md #6).
func DetectDevice(sent Device, userAgent string) Device {
	d := Device{
		Name:   clean(sent.Name),
		OS:     clean(sent.OS),
		Client: clean(sent.Client),
		Source: SourceClient,
	}
	if d.Name != "" || d.OS != "" || d.Client != "" {
		return d
	}

	ua := useragent.New(userAgent)
	browser, version := ua.Browser()
	return Device{
		Name:   clean(ua.Model()),
		OS:     clean(ua.OS()),
		Client: clean(browser + " " + version),
		Source: SourceUserAgent,
	}
}

func clean(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > maxDeviceFieldLength {
		// Cut on a character boundary, so a multi-byte character is not split.
		n := maxDeviceFieldLength
		for n > 0 && !utf8.RuneStart(s[n]) {
			n--
		}
		s = s[:n]
	}
	return s
}
