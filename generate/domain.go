package generate

type Domain interface {
	Next() bool
	Value() string
	Reset()
}
