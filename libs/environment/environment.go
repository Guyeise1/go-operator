package environment

import (
	"fmt"
	"os"
	"strconv"
)

type EnvironmentVariables struct {
	GoApiURL             string
	ControllerNamespace  string
	SecretPrefix         string
	CleanIntervalSeconds int
	Password             string
}

var variables *EnvironmentVariables = nil

func GetVariables() *EnvironmentVariables {
	if variables == nil {
		variables = &EnvironmentVariables{
			GoApiURL:             getenvOrDie("GO_API_SERVER"),
			ControllerNamespace:  getenvOrDie("CONTROLLER_NAMESPACE"),
			SecretPrefix:         getenv("SECRET_PREFIX", "go-"),
			CleanIntervalSeconds: getenvInt("CLEAN_INTERVAL_SECONDS", 15*60),
			Password:             getenvOrDie("PASSWORD"),
		}
	}
	return variables
}

func getenvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	} else {
		ret, err := strconv.Atoi(value)
		if err != nil {
			fmt.Printf("[ERROR - getenvInt] failed to read value for %s, %s cannot be converted to int", key, value)
			return fallback
		} else {
			return ret
		}
	}
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

func getenvOrDie(key string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		panic("Environment variable " + key + "is not defined")
	}
	return value
}
