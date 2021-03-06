package jstore

type Option interface{}
type StoreOption interface{}

func Eq(property string, value interface{}) Option {
	return CompareOption{property, "=", value}
}

func Gt(property string, value interface{}) Option {
	return CompareOption{property, ">", value}
}

func Gte(property string, value interface{}) Option {
	return CompareOption{property, ">=", value}
}

func Lt(property string, value interface{}) Option {
	return CompareOption{property, "<", value}
}

func Lte(property string, value interface{}) Option {
	return CompareOption{property, "<=", value}
}

type CompareOption struct {
	Property  string
	Operation string
	Value     interface{}
}

func Id(value string) Option {
	return IdOption{value}
}

type IdOption struct {
	Value string
}

func SortBy(property string, ascending bool) Option {
	return SortOption{
		property,
		ascending,
	}
}

type SortOption struct {
	Property  string
	Ascending bool
}
