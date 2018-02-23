package jstore

import (
	"encoding/json"
	"errors"
)

type Store interface {
	Delete(project, documentType, id string, options ...Option) error
	Save(project, documentType, id string, json string, options ...Option) error
	Find(project, documentType string, options ...Option) (string, error)
	FindN(project, documentType string, maxResults int, options ...Option) ([]string, error)
}

type JStore interface {
	Store
	Marshal(object interface{}, project, documentType, id string, options ...Option) error
	Unmarshal(objectRef interface{}, project, documentType string, options ...Option) error
	Bucket(project, documentType string) Bucket
}

type Bucket interface {
	Delete(id string, options ...Option) error
	Save(id string, json string, options ...Option) error
	Find(options ...Option) (string, error)
	FindN(maxResults int, options ...Option) ([]string, error)
	Marshal(object interface{}, id string, options ...Option) error
	Unmarshal(objectRef interface{}, options ...Option) error
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

func (store *marshalStore) Marshal(object interface{}, project, documentType, id string, options ...Option) error {
	j, err := json.Marshal(object)
	if err != nil {
		return err
	}
	return store.Save(project, documentType, id, string(j), options...)
}

func (store *marshalStore) Unmarshal(objectRef interface{}, project, documentType string, options ...Option) error {
	j, err := store.Find(project, documentType, options...)
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

func (b *bucket) Delete(id string, options ...Option) error {
	return b.store.Delete(b.project, b.documentType, id, options...)
}

func (b *bucket) Save(id string, json string, options ...Option) error {
	return b.store.Save(b.project, b.documentType, id, json, options...)
}

func (b *bucket) Find(options ...Option) (string, error) {
	return b.store.Find(b.project, b.documentType, options...)
}

func (b *bucket) FindN(maxResults int, options ...Option) ([]string, error) {
	return b.store.FindN(b.project, b.documentType, maxResults, options...)
}

func (b *bucket) Marshal(object interface{}, id string, options ...Option) error {
	return b.store.Marshal(object, b.project, b.documentType, id, options...)
}

func (b *bucket) Unmarshal(objectRef interface{}, options ...Option) error {
	return b.store.Unmarshal(objectRef, b.project, b.documentType, options...)
}
