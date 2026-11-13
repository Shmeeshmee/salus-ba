package config

import (
	"errors"
	"net/url"
	"os"
	"strings"
)

// SMS is the text messages' engine (docs/plans/appointments.md, ADR 0015): the fake (pkg/sms/smstest) unless the
// generic HTTPS adapter (pkg/sms/smshttp) is configured with its credentials. No binary loads it yet: the reminders
// (SAL-33) wire it, and the adapter stays off until the owner chooses an EU/EEA provider, its DPA is recorded in
// docs/privacy/processors.md, and the adapter is checked against its API.
type SMS struct {
	Engine string // SMS_ENGINE: fake (the default) or http
	URL    string // SMS_HTTP_URL: the provider's https endpoint, for http
	Token  string // read from SMS_HTTP_TOKEN_FILE, for http
}

// The engines LoadSMS takes.
const (
	SMSFake = "fake"
	SMSHTTP = "http"
)

// LoadSMS reads the SMS engine. http needs SMS_HTTP_URL (https, no credentials in it) and SMS_HTTP_TOKEN_FILE (a
// readable, non-empty file); either set without http is refused, as is any other engine. Errors name the variable,
// never a value or a path.
func LoadSMS() (SMS, error) {
	engine := os.Getenv("SMS_ENGINE")
	if engine == "" {
		engine = SMSFake
	}
	endpoint, tokenFile := os.Getenv("SMS_HTTP_URL"), os.Getenv("SMS_HTTP_TOKEN_FILE")
	switch engine {
	case SMSFake:
		if endpoint != "" || tokenFile != "" {
			return SMS{}, errors.New("config: SMS_HTTP_URL or SMS_HTTP_TOKEN_FILE is set without SMS_ENGINE=http")
		}
		return SMS{Engine: SMSFake}, nil
	case SMSHTTP:
	default:
		return SMS{}, errors.New("config: SMS_ENGINE is not fake or http")
	}
	u, err := url.Parse(endpoint)
	if endpoint == "" || err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
		return SMS{}, errors.New("config: SMS_ENGINE=http needs SMS_HTTP_URL, an https URL without credentials")
	}
	if tokenFile == "" {
		return SMS{}, errors.New("config: SMS_ENGINE=http needs SMS_HTTP_TOKEN_FILE")
	}
	token, err := readKeyFile("SMS_HTTP_TOKEN_FILE", tokenFile)
	if err != nil {
		return SMS{}, err
	}
	return SMS{Engine: SMSHTTP, URL: endpoint, Token: strings.TrimSpace(string(token))}, nil
}
