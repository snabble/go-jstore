package jstore

import (
	"errors"
)

var (
	NotFound = errors.New("Document not found")
)

func NewStore(driverName, dataSourceName string) (JStore, error) {
	p, found := getProvider(driverName)
	if !found {
		return nil, errors.New("No jstore provider for type: " + driverName)
	}
	return p(dataSourceName)
}

type Store interface {
	Delete(project, documentType, id string) error
	Save(project, documentType, id string, json string) error
	Find(project, documentType string, matcher ...Matcher) (string, error)
	FindN(project, documentType string, maxCount int, matcher ...Matcher) ([]string, error)
}

type MarshalStore interface {
	Marshal(object interface{}, project, documentType, id string) error
	Unmarshal(objectRef interface{}, project, documentType string, matcher ...Matcher) error
}

type JStore interface {
	Store
	MarshalStore
}
