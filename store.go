package jstore

import (
	"encoding/json"
	"errors"
)

type Store interface {
	Delete(project, documentType, id string) error
	Save(project, documentType, id string, json string) error
	Find(project, documentType string, matcher ...Matcher) (string, error)
	FindN(project, documentType string, maxCount int, matcher ...Matcher) ([]string, error)
}

type JStore interface {
	Store
	Marshal(object interface{}, project, documentType, id string) error
	Unmarshal(objectRef interface{}, project, documentType string, matcher ...Matcher) error
	Bucket(project, documentType string) Bucket
}

type Bucket interface {
	Delete(id string) error
	Save(id string, json string) error
	Find(matcher ...Matcher) (string, error)
	FindN(maxCount int, matcher ...Matcher) ([]string, error)
	Marshal(object interface{}, id string) error
	Unmarshal(objectRef interface{}, matcher ...Matcher) error
}

var (
	NotFound = errors.New("Document not found")
)

func NewStore(driverName, dataSourceName string) (JStore, error) {
	p, found := getProvider(driverName)
	if !found {
		return nil, errors.New("No jstore provider for type: " + driverName)
	}
	store, err := p(dataSourceName)
	return &marshalStore{
		Store: store,
	}, err
}

func NewBucket(driverName, dataSourceName, project, documentType string) (Bucket, error) {
	p, found := getProvider(driverName)
	if !found {
		return nil, errors.New("No jstore provider for type: " + driverName)
	}
	store, err := p(dataSourceName)
	if err != nil {
		return nil, err
	}

	marshalStore := &marshalStore{
		Store: store,
	}
	return marshalStore.Bucket(project, documentType), nil
}

type marshalStore struct {
	Store
}

func (store *marshalStore) Marshal(object interface{}, project, documentType, id string) error {
	j, err := json.Marshal(object)
	if err != nil {
		return err
	}
	return store.Save(project, documentType, id, string(j))
}

func (store *marshalStore) Unmarshal(objectRef interface{}, project, documentType string, matcher ...Matcher) error {
	j, err := store.Find(project, documentType, matcher...)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(j), objectRef)
}

func (store *marshalStore) Bucket(project, documentType string) Bucket {
	return &bucket{
		store:        store,
		project:      project,
		documentType: documentType,
	}
}

type bucket struct {
	store        JStore
	project      string
	documentType string
}

func (b *bucket) Delete(id string) error {
	return b.store.Delete(b.project, b.documentType, id)
}

func (b *bucket) Save(id string, json string) error {
	return b.store.Save(b.project, b.documentType, id, json)
}

func (b *bucket) Find(matcher ...Matcher) (string, error) {
	return b.store.Find(b.project, b.documentType, matcher...)
}

func (b *bucket) FindN(maxCount int, matcher ...Matcher) ([]string, error) {
	return b.store.FindN(b.project, b.documentType, maxCount, matcher...)
}

func (b *bucket) Marshal(object interface{}, id string) error {
	return b.store.Marshal(object, b.project, b.documentType, id)
}

func (b *bucket) Unmarshal(objectRef interface{}, matcher ...Matcher) error {
	return b.store.Unmarshal(objectRef, b.project, b.documentType, matcher...)
}
