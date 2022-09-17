package environment

import (
	"os"
)

type EnvironmentVariables struct {
	GoApiURL            string
	ControllerNamespace string
	SecretPrefix        string
}

var Variables = EnvironmentVariables{
	GoApiURL:            os.Getenv("GO_API_SERVER"),
	ControllerNamespace: os.Getenv("CONTROLLER_NAMESPACE"),
	SecretPrefix:        getenv("SECRET_PREFIX", "go-"),
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}
