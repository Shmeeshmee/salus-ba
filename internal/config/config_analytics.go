package config

import (
	"errors"
	"os"
	"strconv"
)

// Analytics holds the analytics settings (docs/plans/analytics.md §2a, A4), read once at start by every binary.
type Analytics struct {
	// PlatformEnabled turns the All of Salus view and its builds on (ANALYTICS_PLATFORM_ENABLED, true or false,
	// false by default). It stays off until the owner confirms the legal answers L1 to L8.
	PlatformEnabled bool
	// MinCount is k of the platform figures (ANALYTICS_MIN_COUNT, 10 by default; below 10 is refused).
	MinCount int
}

// minCount is the smallest k the platform figures accept (A4).
const minCount = 10

func loadAnalytics() (Analytics, error) {
	a := Analytics{MinCount: minCount}
	switch os.Getenv("ANALYTICS_PLATFORM_ENABLED") {
	case "", "false":
	case "true":
		a.PlatformEnabled = true
	default:
		return Analytics{}, errors.New("config: ANALYTICS_PLATFORM_ENABLED is not true or false")
	}
	if v := os.Getenv("ANALYTICS_MIN_COUNT"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < minCount {
			return Analytics{}, errors.New("config: ANALYTICS_MIN_COUNT is not a whole number of at least 10")
		}
		a.MinCount = n
	}
	return a, nil
}
