package config

import "os"

// IsDev reports whether this process runs on the compose stack of a developer's machine: SALUS_ENV is exactly EnvDev,
// which deployments/docker-compose.yml sets and production's env file does not. The demo seed (salusctl demo-seed)
// refuses to run otherwise.
func IsDev() bool { return os.Getenv("SALUS_ENV") == EnvDev }
