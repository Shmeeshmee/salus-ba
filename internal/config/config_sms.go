package config

import (
	"errors"
	"os"
)

// SMS is the text messages' engine (docs/plans/appointments.md, ADR 0015): the fake (pkg/sms/smstest) only. No
// provider is chosen (Q2): its adapter, and the engine and credentials that turn it on, come with the chosen EU/EEA
// provider once its DPA is recorded in docs/privacy/processors.md. No binary loads it before SAL-33.
type SMS struct {
	Engine string // SMS_ENGINE: fake, the default and the only one
}

// SMSFake is the one engine LoadSMS takes.
const SMSFake = "fake"

// LoadSMS reads the SMS engine and refuses any but the fake. Errors name the variable, never a value.
func LoadSMS() (SMS, error) {
	switch os.Getenv("SMS_ENGINE") {
	case "", SMSFake:
		return SMS{Engine: SMSFake}, nil
	}
	return SMS{}, errors.New("config: SMS_ENGINE is not fake; no SMS provider is chosen yet (docs/plans/appointments.md)")
}
