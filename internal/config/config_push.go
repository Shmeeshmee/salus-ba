package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

// Push holds the mobile push services' credentials (docs/deploy/push.md). Load never reads them; only cmd/notifier
// calls LoadPush. Each platform is off until its credentials are set, and then its notifications go through the
// fake (pkg/push/pushtest), which sends nothing. The keys come from files, the root-only key files on the encrypted
// volume in production (ADR 0011), never from a variable's value, the env file or a GitHub secret.
type Push struct {
	// APNs is Apple's push service, for iOS; nil when it is off.
	APNs *APNs
	// FCMCredentials is the Firebase service account's JSON key (the file FCM_CREDENTIALS_FILE names), for
	// Android; nil when FCM is off.
	FCMCredentials []byte
}

// APNs is the team's APNs auth key and the ids it is used with.
type APNs struct {
	KeyID    string // APNS_KEY_ID, the key's Key ID
	TeamID   string // APNS_TEAM_ID, the Apple Developer team's Team ID
	BundleID string // APNS_BUNDLE_ID, the patient app's bundle id (the notifications' topic)
	Key      []byte // the .p8 auth key (the file APNS_KEY_FILE names)
	Sandbox  bool   // APNS_SANDBOX: the tokens are a development build's (false unless "true")
}

// LoadPush reads the push credentials. A platform's settings are all set or none: a half configuration is refused,
// as is APNS_SANDBOX without APNs. Errors name the variable, never a value or a path.
func LoadPush() (Push, error) {
	var p Push
	var err error
	if path := os.Getenv("FCM_CREDENTIALS_FILE"); path != "" {
		if p.FCMCredentials, err = readKeyFile("FCM_CREDENTIALS_FILE", path); err != nil {
			return Push{}, err
		}
	}
	vars := []string{"APNS_KEY_FILE", "APNS_KEY_ID", "APNS_TEAM_ID", "APNS_BUNDLE_ID"}
	var set, missing []string
	for _, v := range vars {
		if os.Getenv(v) == "" {
			missing = append(missing, v)
		} else {
			set = append(set, v)
		}
	}
	sandbox := os.Getenv("APNS_SANDBOX")
	switch {
	case len(set) == 0 && sandbox != "":
		return Push{}, errors.New("config: APNS_SANDBOX is set without APNs; set APNS_KEY_FILE, APNS_KEY_ID, APNS_TEAM_ID and APNS_BUNDLE_ID")
	case len(set) == 0:
		return p, nil
	case len(missing) > 0:
		return Push{}, errors.New("config: APNs is half configured: " + strings.Join(missing, ", ") + " missing")
	}
	a := APNs{KeyID: os.Getenv("APNS_KEY_ID"), TeamID: os.Getenv("APNS_TEAM_ID"), BundleID: os.Getenv("APNS_BUNDLE_ID")}
	if a.Key, err = readKeyFile("APNS_KEY_FILE", os.Getenv("APNS_KEY_FILE")); err != nil {
		return Push{}, err
	}
	if sandbox != "" {
		if a.Sandbox, err = strconv.ParseBool(sandbox); err != nil {
			return Push{}, errors.New("config: APNS_SANDBOX is not true or false")
		}
	}
	p.APNs = &a
	return p, nil
}

// readKeyFile reads the key file at path, which the variable name gives; an unreadable or blank file is an error
// naming the variable only.
func readKeyFile(name, path string) ([]byte, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.New("config: " + name + " cannot be read")
	}
	if strings.TrimSpace(string(b)) == "" {
		return nil, errors.New("config: " + name + " is empty")
	}
	return b, nil
}
