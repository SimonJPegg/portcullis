package domain

import "errors"

type Verdict interface {
	isVerdict()
}

type Allowed struct {}

type Denied struct {
	Reason string
	PolicyId *string
}

type Pending struct {}

func (p Pending) isVerdict() {}
func (a Allowed) isVerdict() {}
func (d Denied) isVerdict() {}

func NewDenied(r string, p *string) (Denied, error) {
	if r == "" {
		return Denied{}, errors.New("a reason for denial must be specified")
	}
	return Denied{r, p}, nil
}
