package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPackageCoordinate_Valid(t *testing.T) {
	pkg, err := NewPackageCoordinate(Maven, "org.antipathy.sluice", "1.0.0")
	assert.NoError(t, err)
	assert.Equal(t, pkg.Name(), "org.antipathy.sluice")
	assert.Equal(t, pkg.EcoSystem(), Maven)
}

func TestPackageCoordinate_EmptyName(t *testing.T) {
	pkg, err := NewPackageCoordinate(Maven, "", "1.0.0")
	assert.Error(t, err)
	assert.Equal(t, PackageCoordinate{}, pkg)
}

func TestPackageCoordinate_EmptyVersion(t *testing.T) {
	pkg, err := NewPackageCoordinate(Maven, "org.antipathy.sluice", "")
	assert.Error(t, err)
	assert.Equal(t, PackageCoordinate{}, pkg)
}
