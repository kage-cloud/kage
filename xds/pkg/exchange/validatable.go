package exchange

type Validatable interface {
	Validate() error
}
