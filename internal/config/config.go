// Package config is the only package that reads the environment.
package config

import (
	"encoding/base64"
	"errors"
	"net"
	"net/netip"
	"net/url"
	"os"
	"strings"
)

// Config holds the settings the binaries need.
type Config struct {
	DatabaseURL string
	HTTPAddr    string
	RedisURL    string // only the binaries that use Redis require it (redis.Open rejects an empty URL)
	// AppDatabasePassword is set on the salus_app role by `migrate up` when not empty. Production leaves it
	// unset when the role is provisioned outside migrate.
	AppDatabasePassword string
	// WebBaseURL is the web app's origin, for links in e-mails (WEB_BASE_URL).
	WebBaseURL string
	// PatientAppURL is the patient app's link base, for appointment e-mails (PATIENT_APP_URL): they link to
	// <PatientAppURL>/appointments. The notifier requires it; its SMTP server is LoadSMTP's.
	PatientAppURL string
	// FHIRBaseURL is the FHIR API's absolute public base, https://…/fhir/R4, for the links and fullUrls of
	// a search's Bundle (FHIR_BASE_URL; the api requires it).
	FHIRBaseURL string
	// TOTPKey seals TOTP secrets (TOTP_KEY, base64, 32 bytes; or the file TOTP_KEY_FILE names, as production
	// mounts it from the encrypted volume: keySetting).
	TOTPKey []byte
	// RateLimitKey keys the HMAC that names the login rate-limit counters (RATE_LIMIT_KEY, base64).
	RateLimitKey []byte
	// WebhookKey seals the clinics' webhook secrets (WEBHOOK_KEY, base64, 32 bytes; or the file WEBHOOK_KEY_FILE
	// names, as production mounts it from the encrypted volume: keySetting). The api and the worker require it.
	WebhookKey []byte
	// TrustedProxies may set X-Forwarded-For (TRUSTED_PROXY_CIDRS, comma-separated CIDRs).
	TrustedProxies []netip.Prefix
}

// Load reads the configuration from the environment. Errors name the variable, never a value.
func Load() (Config, error) {
	cfg := Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		HTTPAddr:    os.Getenv("HTTP_ADDR"),
		RedisURL:    os.Getenv("REDIS_URL"),

		AppDatabasePassword: os.Getenv("APP_DATABASE_PASSWORD"),
		WebBaseURL:          os.Getenv("WEB_BASE_URL"),
		PatientAppURL:       os.Getenv("PATIENT_APP_URL"),
		FHIRBaseURL:         os.Getenv("FHIR_BASE_URL"),
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("config: DATABASE_URL is required")
	}
	if cfg.HTTPAddr == "" {
		cfg.HTTPAddr = ":8080"
	}
	if cfg.FHIRBaseURL != "" && !strings.HasSuffix(strings.TrimSuffix(cfg.FHIRBaseURL, "/"), fhirPath) {
		return Config{}, errors.New("config: FHIR_BASE_URL does not end in " + fhirPath)
	}
	totpKey, err := keySetting("TOTP_KEY")
	if err != nil {
		return Config{}, err
	}
	if cfg.TOTPKey, err = base64.StdEncoding.DecodeString(totpKey); err != nil {
		return Config{}, errors.New("config: TOTP_KEY is not base64")
	}
	webhookKey, err := keySetting("WEBHOOK_KEY")
	if err != nil {
		return Config{}, err
	}
	if cfg.WebhookKey, err = base64.StdEncoding.DecodeString(webhookKey); err != nil {
		return Config{}, errors.New("config: WEBHOOK_KEY is not base64")
	}
	if cfg.RateLimitKey, err = base64.StdEncoding.DecodeString(os.Getenv("RATE_LIMIT_KEY")); err != nil {
		return Config{}, errors.New("config: RATE_LIMIT_KEY is not base64")
	}
	for _, s := range strings.Split(os.Getenv("TRUSTED_PROXY_CIDRS"), ",") {
		if s = strings.TrimSpace(s); s == "" {
			continue
		}
		p, err := netip.ParsePrefix(s)
		if err != nil {
			return Config{}, errors.New("config: TRUSTED_PROXY_CIDRS holds an entry that is not a CIDR")
		}
		cfg.TrustedProxies = append(cfg.TrustedProxies, p)
	}
	return cfg, nil
}

// fhirPath ends FHIR_BASE_URL; the rest is the api's public base.
const fhirPath = "/fhir/R4"

// APIBaseURL is the api's absolute public base, FHIR_BASE_URL without /fhir/R4, for the download URL of a
// valueAttachment (docs/attachments/SPEC.md §2 "Configuration"); empty when FHIR_BASE_URL is.
func (c Config) APIBaseURL() string {
	return strings.TrimSuffix(strings.TrimSuffix(c.FHIRBaseURL, "/"), fhirPath)
}

// Retention holds the retention job's credentials: the salus_retention role, which bypasses row-level
// security and may delete the rows whose period is over (docs/booking/SPEC.md §2 "Retention privileges").
// Load never reads them; only cmd/worker (the pool of the retention_apply job) and cmd/migrate (which sets
// the role's password) call LoadRetention.
type Retention struct {
	// DatabaseURL is the role's login (RETENTION_DATABASE_URL); cmd/worker requires it.
	DatabaseURL string
	// Password is set on the role by `migrate up` when not empty (RETENTION_DATABASE_PASSWORD), like
	// AppDatabasePassword.
	Password string
}

// LoadRetention reads the retention job's credentials from the environment.
func LoadRetention() Retention {
	return Retention{
		DatabaseURL: os.Getenv("RETENTION_DATABASE_URL"),
		Password:    os.Getenv("RETENTION_DATABASE_PASSWORD"),
	}
}

// Stats holds the aggregate builds' credentials: the salus_stats role, which bypasses row-level security and
// reads only the source columns of the analytics (docs/plans/analytics.md, ADR 0013). Load never reads them;
// cmd/migrate (which sets the role's password) and cmd/worker (the builds' pool) call LoadStats.
type Stats struct {
	// DatabaseURL is the role's login (STATS_DATABASE_URL); cmd/worker requires it.
	DatabaseURL string
	// Password is set on the role by `migrate up` when not empty (STATS_DATABASE_PASSWORD).
	Password string
}

// LoadStats reads the aggregate builds' credentials from the environment.
func LoadStats() Stats {
	return Stats{
		DatabaseURL: os.Getenv("STATS_DATABASE_URL"),
		Password:    os.Getenv("STATS_DATABASE_PASSWORD"),
	}
}

// LiveKit holds the video server's settings (docs/consultation/SPEC.md §2 "Configuration"). Load never
// reads them; only cmd/api and cmd/worker call LoadLiveKit. The key pair is synthetic in compose and comes
// from the secret store in production, never from a committed file.
type LiveKit struct {
	URL       string // the server API, as the api and the worker reach it (LIVEKIT_URL)
	PublicURL string // what the clients connect to (LIVEKIT_PUBLIC_URL)
	APIKey    string // LIVEKIT_API_KEY
	APISecret string // LIVEKIT_API_SECRET
}

// LoadLiveKit reads the LiveKit settings from the environment; each is required. Errors name the variable,
// never a value.
func LoadLiveKit() (LiveKit, error) {
	lk := LiveKit{
		URL:       os.Getenv("LIVEKIT_URL"),
		PublicURL: os.Getenv("LIVEKIT_PUBLIC_URL"),
		APIKey:    os.Getenv("LIVEKIT_API_KEY"),
		APISecret: os.Getenv("LIVEKIT_API_SECRET"),
	}
	for _, v := range []struct{ name, value string }{
		{"LIVEKIT_URL", lk.URL}, {"LIVEKIT_PUBLIC_URL", lk.PublicURL},
		{"LIVEKIT_API_KEY", lk.APIKey}, {"LIVEKIT_API_SECRET", lk.APISecret},
	} {
		if v.value == "" {
			return LiveKit{}, errors.New("config: " + v.name + " is required")
		}
	}
	return lk, nil
}

// ObjectStore holds the S3-compatible object store's settings and the key-encryption key that wraps each
// file's data key (docs/attachments/SPEC.md §2 "Configuration", "Encryption"). Load never reads them; only
// cmd/api and cmd/worker call LoadObjectStore. Locally the store is the compose Garage and the KEK is
// synthetic; production takes the KEK from a key file on the encrypted volume (BINARY_KEK_FILE, ADR 0011), never
// from a committed file or the env file.
type ObjectStore struct {
	Endpoint        string // S3_ENDPOINT, an absolute http(s) URL
	Region          string // S3_REGION
	Bucket          string // S3_BUCKET
	AccessKeyID     string // S3_ACCESS_KEY_ID
	SecretAccessKey string // S3_SECRET_ACCESS_KEY
	KEKID           string // BINARY_KEK_ID, stored with each wrapped data key
	KEK             []byte // BINARY_KEK (or the file BINARY_KEK_FILE names: keySetting), base64 of 32 bytes (AES-256)
}

// LoadObjectStore reads the object store's settings from the environment; each is required. Errors name the
// variable, never a value.
func LoadObjectStore() (ObjectStore, error) {
	s := ObjectStore{
		Endpoint:        os.Getenv("S3_ENDPOINT"),
		Region:          os.Getenv("S3_REGION"),
		Bucket:          os.Getenv("S3_BUCKET"),
		AccessKeyID:     os.Getenv("S3_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("S3_SECRET_ACCESS_KEY"),
		KEKID:           os.Getenv("BINARY_KEK_ID"),
	}
	kek, err := keySetting("BINARY_KEK")
	if err != nil {
		return ObjectStore{}, err
	}
	for _, v := range []struct{ name, value string }{
		{"S3_ENDPOINT", s.Endpoint}, {"S3_REGION", s.Region}, {"S3_BUCKET", s.Bucket},
		{"S3_ACCESS_KEY_ID", s.AccessKeyID}, {"S3_SECRET_ACCESS_KEY", s.SecretAccessKey},
		{"BINARY_KEK", kek}, {"BINARY_KEK_ID", s.KEKID},
	} {
		if v.value == "" {
			return ObjectStore{}, errors.New("config: " + v.name + " is required")
		}
	}
	if u, err := url.Parse(s.Endpoint); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return ObjectStore{}, errors.New("config: S3_ENDPOINT is not an absolute http(s) URL")
	}
	if s.KEK, err = base64.StdEncoding.DecodeString(kek); err != nil || len(s.KEK) != 32 {
		return ObjectStore{}, errors.New("config: BINARY_KEK is not base64 of 32 bytes")
	}
	return s, nil
}

// keySetting reads a key from the variable name, or from the file the variable name_FILE names: production
// mounts TOTP_KEY, BINARY_KEK and WEBHOOK_KEY as compose secrets from a root-only directory on the encrypted volume, so they
// never sit in the env file (deployments/production/README.md "Keys"). Locally and in CI the variable is set.
// Setting both is an error, and so is a file that cannot be read or holds nothing but whitespace; the
// whitespace around the value (the file's newline) is dropped. Errors name the variable, never a value or path.
func keySetting(name string) (string, error) {
	value, path := os.Getenv(name), os.Getenv(name+"_FILE")
	if path == "" {
		return value, nil
	}
	if value != "" {
		return "", errors.New("config: " + name + " and " + name + "_FILE are both set; set one")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", errors.New("config: " + name + "_FILE cannot be read")
	}
	value = strings.TrimSpace(string(b))
	if value == "" {
		return "", errors.New("config: " + name + "_FILE is empty")
	}
	return value, nil
}

// Clamd holds the malware scanner's address (docs/attachments/SPEC.md §2 "Configuration"). Only cmd/worker
// calls LoadClamd.
type Clamd struct {
	Addr string // CLAMD_ADDR, host:port of clamd's TCP socket
}

// LoadClamd reads clamd's address from the environment; it is required. Errors name the variable, never a
// value.
func LoadClamd() (Clamd, error) {
	addr := os.Getenv("CLAMD_ADDR")
	if host, port, err := net.SplitHostPort(addr); err != nil || host == "" || port == "" {
		return Clamd{}, errors.New("config: CLAMD_ADDR is not host:port")
	}
	return Clamd{Addr: addr}, nil
}

// STTEngine names a speech-to-text engine of the transcription agent (cmd/agent).
type STTEngine string

// The engines. The fake is the default: it plays placeholder captions and sends no audio anywhere. Azure is a
// processor (docs/privacy/processors.md) and is not enabled with real patient audio until its DPA, an EU region
// and zero data retention are in writing (docs/plans/consult-intelligence.md "Gates").
const (
	STTFake  STTEngine = "fake"
	STTAzure STTEngine = "azure"
)

// STT holds the transcription agent's speech-to-text settings. Load never reads them; only cmd/agent calls
// LoadSTT. The Azure key comes from the secret store, never from a committed file.
type STT struct {
	Engine      STTEngine // STT_ENGINE: fake (the default) or azure
	AzureRegion string    // AZURE_SPEECH_REGION, required for azure; pkg/stt/azure accepts EU regions only
	AzureKey    string    // AZURE_SPEECH_KEY, required for azure
	Script      string    // FAKE_ENGINE_SCRIPT, for the fake only (fakeScript)
}

// fakeScript is FAKE_ENGINE_SCRIPT: the directory of the scripted conversations the fake engines play (stttest
// and llmtest LoadScript), set only by make e2e's compose file (deployments/docker-compose.e2e.yml). The compose
// default and production leave it unset; beside a real engine it is refused, so a script never stands in for
// one and test lines never reach one.
func fakeScript() string {
	return os.Getenv("FAKE_ENGINE_SCRIPT")
}

var errScriptBesideARealEngine = errors.New("config: FAKE_ENGINE_SCRIPT is for the fake engines only")

// LoadSTT reads the speech-to-text settings from the environment. Errors name the variable, never a value.
func LoadSTT() (STT, error) {
	s := STT{Engine: STTEngine(os.Getenv("STT_ENGINE"))}
	switch s.Engine {
	case "", STTFake:
		return STT{Engine: STTFake, Script: fakeScript()}, nil
	case STTAzure:
	default:
		return STT{}, errors.New("config: STT_ENGINE is neither fake nor azure")
	}
	if fakeScript() != "" {
		return STT{}, errScriptBesideARealEngine
	}
	s.AzureRegion, s.AzureKey = os.Getenv("AZURE_SPEECH_REGION"), os.Getenv("AZURE_SPEECH_KEY")
	if s.AzureRegion == "" {
		return STT{}, errors.New("config: AZURE_SPEECH_REGION is required for STT_ENGINE=azure")
	}
	if s.AzureKey == "" {
		return STT{}, errors.New("config: AZURE_SPEECH_KEY is required for STT_ENGINE=azure")
	}
	return s, nil
}

// LLMEngine names the language model of the consultation pipeline: translation (SAL-5), suggestions (SAL-7)
// and drafting (SAL-8), the pkg/llm ports.
type LLMEngine string

// The engines. The fake is the default: it sends no text anywhere. Anthropic's Claude API (pkg/llm/anthropic) is
// a processor (docs/privacy/processors.md) and is not enabled with real patient data until its DPA, zero data
// retention and the plan's EU region condition are met (docs/plans/consult-intelligence.md "Gates").
const (
	LLMFake      LLMEngine = "fake"
	LLMAnthropic LLMEngine = "anthropic"
)

// defaultLLMModel is the Claude model when LLM_MODEL is not set.
const defaultLLMModel = "claude-opus-5-5"

// LLM holds the language model's settings. Load never reads them; only a command's main calls LoadLLM. The key
// comes from the secret store, never from a committed file.
type LLM struct {
	Engine       LLMEngine // LLM_ENGINE: fake (the default) or anthropic
	AnthropicKey string    // ANTHROPIC_API_KEY, required for anthropic
	Model        string    // LLM_MODEL, for anthropic: claude-opus-5-5 when unset
	Script       string    // FAKE_ENGINE_SCRIPT, for the fake only (fakeScript)
}

// LoadLLM reads the language model's settings from the environment; for the fake, none but the engine. Errors
// name the variable, never a value.
func LoadLLM() (LLM, error) {
	switch LLMEngine(os.Getenv("LLM_ENGINE")) {
	case "", LLMFake:
		return LLM{Engine: LLMFake, Script: fakeScript()}, nil
	case LLMAnthropic:
	default:
		return LLM{}, errors.New("config: LLM_ENGINE is neither fake nor anthropic")
	}
	if fakeScript() != "" {
		return LLM{}, errScriptBesideARealEngine
	}
	l := LLM{Engine: LLMAnthropic, AnthropicKey: os.Getenv("ANTHROPIC_API_KEY"), Model: os.Getenv("LLM_MODEL")}
	if l.AnthropicKey == "" {
		return LLM{}, errors.New("config: ANTHROPIC_API_KEY is required for LLM_ENGINE=anthropic")
	}
	if l.Model == "" {
		l.Model = defaultLLMModel
	}
	return l, nil
}
