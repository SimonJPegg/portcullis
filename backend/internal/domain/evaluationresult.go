package domain

import "time"

type EvaluationResult struct {
	PackageCoordinate PackageCoordinate
	PolicyResult []PolicyResult
	EvaluatedAt time.Time
}
