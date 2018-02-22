package jstore

type Matcher interface{}

func Eq(property string, value interface{}) EqMatcher {
	return EqMatcher{property, value}
}

type EqMatcher struct {
	Property string
	Value    interface{}
}

func Id(value string) IdMatcher {

	return IdMatcher{value}
}

type IdMatcher struct {
	Value string
}
