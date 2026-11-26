package config

import (
	"errors"
	"os"
	"slices"
	"strings"
)

// Operator is who answers the data requests to the Salus account (docs/clinic-cms/WAVE5.md S1, D5): where their
// deadline warnings go and in which language. Not secret.
type Operator struct {
	Email    string // SALUS_OPERATOR_EMAIL; "" only in dev
	Language string // SALUS_OPERATOR_LANGUAGE, one of the ten; bs when unset
}

// LoadOperator reads the operator's address and language, for the notifier, which sends the deadline warnings. Without
// an address (or with a placeholder in brackets, as configs/.env.example's [OPERATOR EMAIL]: isPlaceholder) the start-up fails, except with SALUS_ENV=dev, where the warnings
// to the operator are logged by their opaque ids instead. Errors name the variable, never a value.
func LoadOperator() (Operator, error) {
	o := Operator{Email: strings.TrimSpace(os.Getenv("SALUS_OPERATOR_EMAIL")), Language: strings.TrimSpace(os.Getenv("SALUS_OPERATOR_LANGUAGE"))}
	if isPlaceholder(o.Email) {
		o.Email = ""
	}
	if o.Language == "" {
		o.Language = "bs"
	}
	// The BCP 47 tags of the Postgres enum language (CLAUDE.md "Languages").
	if !slices.Contains([]string{"bs", "hr", "sr-Latn", "sr-Cyrl", "en", "de", "ar", "tr", "it", "fr"}, o.Language) {
		return Operator{}, errors.New("config: SALUS_OPERATOR_LANGUAGE is not one of the ten languages")
	}
	if o.Email == "" && os.Getenv("SALUS_ENV") != EnvDev {
		return Operator{}, errors.New("config: SALUS_OPERATOR_EMAIL is required outside SALUS_ENV=dev")
	}
	return o, nil
}
