package casper

type MultiEndpoints interface {
	GetOne() (endpoint string, err error)
}
