package domain

import (
	"errors"
	"fmt"
)

type EcoSystem string

const (
	Maven EcoSystem =  "maven"
	PyPI  EcoSystem = "pypi"
)

func NewEcoSystem(s string) (EcoSystem, error)  {
	if s == "" {
		return "", errors.New("ecoSystem name cannot be empty")
	}

	eco := EcoSystem(s)

	if eco != Maven && eco != PyPI {
		return "", fmt.Errorf("ecoSystem must be one of %s or %s", Maven, PyPI)
	}
	return eco, nil
}
