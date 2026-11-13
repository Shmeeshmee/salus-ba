package config

import (
	"errors"
	"os"
)

// SMS is the text messages' engine (docs/plans/appointments.md, ADR 0015): the fake (pkg/sms/smstest) only. No
// provider is chosen (Q2): its adapter, and the engine and credentials that turn it on, come with the chosen EU/EEA
// provider once its DPA is recorded in docs/privacy/processors.md. No binary loads it before SAL-33.
type SMS struct {
	Engine string // SMS_ENGINE: fake, the default and the only one taken (SMSInfobip is named, not taken)
}

// The engines LoadSMS knows. SMSFake is the one it takes. SMSInfobip is the provider the adapter (an sms.Sender over
// Infobip's API) is planned for: refused until Infobip's DPA is recorded in docs/privacy/processors.md and its
// adapter is written; no request to it is made before then.
const (
	SMSFake    = "fake"
	SMSInfobip = "infobip"
)

// LoadSMS reads the SMS engine and refuses any but the fake. Errors name the variable, never a value.
func LoadSMS() (SMS, error) {
	switch os.Getenv("SMS_ENGINE") {
	case "", SMSFake:
		return SMS{Engine: SMSFake}, nil
	case SMSInfobip:
		return SMS{}, errors.New("config: SMS_ENGINE=infobip waits for Infobip's DPA in docs/privacy/processors.md and its adapter")
	}
	return SMS{}, errors.New("config: SMS_ENGINE is not fake; no SMS provider is chosen yet (docs/plans/appointments.md)")
}
