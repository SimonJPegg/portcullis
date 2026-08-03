package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewEcosystem(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    EcoSystem
		wantErr bool
	}{
		{"valid pypi", "pypi", PyPI, false},
		{"valid maven", "maven", Maven, false},
		{"empty string", "", EcoSystem(""), true},
		{"unsupported ecosystem", "npm", EcoSystem(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewEcoSystem(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
