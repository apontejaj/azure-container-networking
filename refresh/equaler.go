package refresh

type Equaler[T any] interface {
	Equal(T) bool
}
