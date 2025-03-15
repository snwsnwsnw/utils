package server

type IMap[KEY comparable, VALUE any] interface {
	Get(KEY) (VALUE, bool)
	Set(KEY, VALUE)
	Delete(...KEY)
	Len() int
	Range(func(KEY, VALUE) bool)
}
