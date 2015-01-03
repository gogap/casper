package casper

import (
	"math/rand"
)

type RandomEndpoints []string

func (p RandomEndpoints) GetOne() (endpoint string, err error) {
	rand.Seed(2010104)
	return p[rand.Intn(len(p))], nil
}
