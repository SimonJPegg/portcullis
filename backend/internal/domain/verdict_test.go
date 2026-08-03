package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)



func TestVerdict_TypeSwitch(t *testing.T) {
	t.Run("allowed", func(t *testing.T) {
		var v Verdict = Allowed{}

		switch v.(type) {
		case Allowed:
			// expected
		default:
			t.Fatal("expected Allowed")
		}
	})

	t.Run("pending", func(t *testing.T) {
		var v Verdict = Pending{}

		switch v.(type) {
		case Pending:
			// expected
		default:
			t.Fatal("expected Pending")
		}
	})

	t.Run("denied", func(t *testing.T) {
		var v Verdict = Denied{Reason: "too old", PolicyId: nil}

		switch d := v.(type) {
		case Denied:
			assert.Equal(t, "too old", d.Reason)
		default:
			t.Fatal("expected Denied")
		}
	})
}

func TestNewDenied_EmptyString(t *testing.T) {
	v, err := NewDenied("", nil)
	assert.Equal(t, Denied{}, v)
	assert.Error(t, err)
}

func TestNewDenied_ValidString(t *testing.T) {
	v, err := NewDenied("Hello, this is Mr Burns", nil)
	assert.NoError(t, err)
	assert.Equal(t, "Hello, this is Mr Burns", v.Reason)
}

func TestNewDenied_ValidPolicyId(t *testing.T) {
	p := "5"
	v, err := NewDenied("Hello, this is Mr Burns", &p)
	assert.NoError(t, err)
	assert.Equal(t, "Hello, this is Mr Burns", v.Reason)
	assert.Equal(t, &p, v.PolicyId)
}
