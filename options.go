package jstore

type Option interface{}

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

var SyncUpdates = "SyncUpdates"
