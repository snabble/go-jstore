package jstore

import (
	"encoding/json"
	"errors"
)

type EntityID struct {
	Project      string
	DocumentType string
	ID           string
}

func ID(project string, documentType string, id string) EntityID {
	return EntityID{project, documentType, id}
}

type Entity struct {
	EntityID
	JSON string
}

type Store interface {
	Delete(id EntityID) error
	Save(id EntityID, json string) error
	Get(id EntityID) (Entity, error)
	Find(project, documentType string, options ...Option) (Entity, error)
	FindN(project, documentType string, maxResults int, options ...Option) ([]Entity, error)
	HealthCheck() error
}

type JStore interface {
	Store
	Marshal(object interface{}, id EntityID) error
	Unmarshal(objectRef interface{}, project, documentType string, options ...Option) error
	Bucket(project, documentType string) Bucket
}

type Bucket interface {
	Delete(id string) error
	Save(id string, json string) error
	Get(id string) (Entity, error)
	Find(options ...Option) (Entity, error)
	FindN(maxResults int, options ...Option) ([]Entity, error)
	Marshal(object interface{}, id string) error
	Unmarshal(objectRef interface{}, options ...Option) error
}

var (
	NotFound = errors.New("Document not found")
)

func NewStore(driverName, dataSourceName string, options ...StoreOption) (JStore, error) {
	p, found := getProvider(driverName)
	if !found {
		return nil, errors.New("No jstore provider for type: " + driverName)
	}
	store, err := p(dataSourceName, options...)
	return &marshalStore{
		Store: store,
	}, err
}

func NewBucket(driverName, dataSourceName, project, documentType string, options ...StoreOption) (Bucket, error) {
	p, found := getProvider(driverName)
	if !found {
		return nil, errors.New("No jstore provider for type: " + driverName)
	}
	store, err := p(dataSourceName, options...)
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

func (store *marshalStore) Marshal(object interface{}, id EntityID) error {
	j, err := json.Marshal(object)
	if err != nil {
		return err
	}
	return store.Save(id, string(j))
}

func (store *marshalStore) Unmarshal(objectRef interface{}, project, documentType string, options ...Option) error {
	j, err := store.Find(project, documentType, options...)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(j.JSON), objectRef)
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
	return b.store.Delete(b.entityID(id))
}

func (b *bucket) Save(id string, json string) error {
	return b.store.Save(b.entityID(id), json)
}

func (b *bucket) Get(id string) (Entity, error) {
	return b.store.Get(b.entityID(id))
}

func (b *bucket) Find(options ...Option) (Entity, error) {
	return b.store.Find(b.project, b.documentType, options...)
}

func (b *bucket) FindN(maxResults int, options ...Option) ([]Entity, error) {
	return b.store.FindN(b.project, b.documentType, maxResults, options...)
}

func (b *bucket) Marshal(object interface{}, id string) error {
	return b.store.Marshal(object, b.entityID(id))
}

func (b *bucket) Unmarshal(objectRef interface{}, options ...Option) error {
	return b.store.Unmarshal(objectRef, b.project, b.documentType, options...)
}

func (b *bucket) entityID(id string) EntityID {
	return EntityID{
		Project:      b.project,
		DocumentType: b.documentType,
		ID:           id,
	}
}
