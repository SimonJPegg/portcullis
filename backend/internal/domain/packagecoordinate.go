package domain

import "errors"

type PackageCoordinate struct {
	ecoSystem EcoSystem
	name      string
	// Version is treated as a lookup key, not parsed or compared.
	// Ecosystem-specific ordering belongs in policy logic, not the domain model.
	version string
}

func NewPackageCoordinate(e EcoSystem, n string, v string) (PackageCoordinate, error) {

	if n == "" {
		return PackageCoordinate{}, errors.New("name must be defined")
	}
	if  v == "" {
		return PackageCoordinate{}, errors.New("version must be defined")
	}

	return PackageCoordinate{
		ecoSystem: e,
		name: n,
		version: v,
	}, nil
}

func (p PackageCoordinate) Name() string {
	return p.name
}

func (p PackageCoordinate) Version() string {
	return p.version
}

func (p PackageCoordinate) EcoSystem() EcoSystem {
	return p.ecoSystem
}
