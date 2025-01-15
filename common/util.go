package common

import (
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

func PrefixEnvVar(prefix, suffix string) []string {
	return []string{prefix + "_" + suffix}
}

func ValidateEnvVars(prefix string, flag []cli.Flag, log log.Logger) {
	for _, envVar := range validateEnvVars(prefix, os.Environ(), cliFlagsToEnvVars(flags)) {
		log.Warn("Unknown env var", "prefix", prefix, "envVar", envVar)
	}
}

func cliFlagsToEnvVars(flags []cli.Flag) map[string]struct{} {
	definedEnvVars := make(map[string]struct{})
	for _, flag := range flags {
		envVars := reflect.ValueOf(flag).Elem().FieldByName("EnvVars")
		for i := 0; i < envVars.Len(); i++ {
			envVarField := envVars.Index(i)
			definedEnvVars[envVarField.String()] = struct{}{}
		}
	}
	return definedEnvVars
}

func validateEnvVars(prefix string, providedEnvVars []string, definedEnvVars map[string]struct{}) []string {
	var out []string
	for _, envVar := range providedEnvVars {
		parts := strings.Split(envVar, "=")
		if len(parts) == 0 {
			continue
		}
		key := parts[0]
		if strings.HasSuffix(key, prefix) {
			if _, ok := definedEnvVars[key]; !ok {
				out = append(out, envVar)
			}
		}
	}
	return out
}

func ParseAddress(address string) (common.Address, error) {
	if common.IsHexAddress(address) {
		return common.HexToAddress(address), nil
	}
	return common.Address{}, fmt.Errorf("invalid address: %s ", address)
}
