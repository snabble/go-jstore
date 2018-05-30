package jstore

import (
	"encoding/json"
	"errors"
)

const NoVersion = 0

type EntityID struct {
	Project      string
	DocumentType string
	ID           string
	Version      int64
}

func NewID(project string, documentType string, id string) EntityID {
	return EntityID{
		Project:      project,
		DocumentType: documentType,
		ID:           id,
	}
}

func NewIDWithVersion(project string, documentType string, id string, version int64) EntityID {
	return EntityID{
		Project:      project,
		DocumentType: documentType,
		ID:           id,
		Version:      version,
	}
}

type Entity struct {
	EntityID
	ObjectRef interface{}
	JSON      string
}

type Store interface {
	Delete(id EntityID) error
	Save(id EntityID, json string) (EntityID, error)
	Get(id EntityID) (Entity, error)
	Find(project, documentType string, options ...Option) (Entity, error)
	FindN(project, documentType string, maxResults int, options ...Option) ([]Entity, error)
	HealthCheck() error
}

type JStore interface {
	Store
	Marshal(object interface{}, id EntityID) (EntityID, error)
	Unmarshal(entityOrObjectRef interface{}, project, documentType string, options ...Option) error
	Bucket(project, documentType string) Bucket
}

type Bucket interface {
	Delete(id EntityID) error
	Save(id EntityID, json string) (EntityID, error)
	Get(id EntityID) (Entity, error)
	Find(options ...Option) (Entity, error)
	FindN(maxResults int, options ...Option) ([]Entity, error)
	Marshal(object interface{}, id EntityID) (EntityID, error)
	Unmarshal(entityOrObjectRef interface{}, options ...Option) error
}

var (
	NotFound               = errors.New("Document not found")
	OptimisticLockingError = errors.New("Optimistic locking failed")
)

func NewStore(driverName, dataSourceName string, options ...StoreOption) (JStore, error) {
	p, found := getProvider(driverName)
	if !found {
		return nil, errors.New("No jstore provider for type: " + driverName)
	}
	store, err := p(dataSourceName, options...)

	if err != nil {
		return nil, err
	}

	return WrapStore(store), nil
}

func WrapStore(store Store) JStore {
	return &marshalStore{
		Store: store,
	}
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

func (store *marshalStore) Marshal(object interface{}, id EntityID) (EntityID, error) {
	j, err := json.Marshal(object)
	if err != nil {
		return EntityID{}, err
	}
	return store.Save(id, string(j))
}

func (store *marshalStore) Unmarshal(entityOrObjectRef interface{}, project, documentType string, options ...Option) error {
	found, err := store.Find(project, documentType, options...)
	if err != nil {
		return err
	}

	objectRef := entityOrObjectRef
	if entity, ok := entityOrObjectRef.(*Entity); ok {
		entity.EntityID = found.EntityID
		entity.JSON = found.JSON
		objectRef = entity.ObjectRef
	}

	return json.Unmarshal([]byte(found.JSON), objectRef)
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

func (b *bucket) Delete(id EntityID) error {
	return b.store.Delete(b.resolveRelativeToBucket(id))
}

func (b *bucket) Save(id EntityID, json string) (EntityID, error) {
	return b.store.Save(b.resolveRelativeToBucket(id), json)
}

func (b *bucket) Get(id EntityID) (Entity, error) {
	return b.store.Get(b.resolveRelativeToBucket(id))
}

func (b *bucket) Find(options ...Option) (Entity, error) {
	return b.store.Find(b.project, b.documentType, options...)
}

func (b *bucket) FindN(maxResults int, options ...Option) ([]Entity, error) {
	return b.store.FindN(b.project, b.documentType, maxResults, options...)
}

func (b *bucket) Marshal(object interface{}, id EntityID) (EntityID, error) {
	return b.store.Marshal(object, b.resolveRelativeToBucket(id))
}

func (b *bucket) Unmarshal(entityOrObjectRef interface{}, options ...Option) error {
	return b.store.Unmarshal(entityOrObjectRef, b.project, b.documentType, options...)
}

func (b *bucket) resolveRelativeToBucket(id EntityID) EntityID {
	return EntityID{
		Project:      b.project,
		DocumentType: b.documentType,
		ID:           id.ID,
		Version:      id.Version,
	}
}
