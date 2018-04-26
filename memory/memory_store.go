package memory

import (
	"sync"

	"encoding/json"

	"github.com/pkg/errors"
	"github.com/snabble/go-jstore"
)

var DriverName = "memory"

func init() {
	jstore.RegisterProvider(DriverName, NewMemoryStore)
}

type storageItem struct {
	entity jstore.Entity
	object map[string]interface{}
}

func newItem(entity jstore.Entity) (storageItem, error) {
	item := storageItem{
		entity: entity,
	}
	err := json.Unmarshal([]byte(entity.JSON), &item.object)
	if err != nil {
		return item, errors.Wrap(err, "could not unmarshall json")
	}
	return item, nil
}

func (item *storageItem) matches(options ...jstore.Option) (bool, error) {
	result := true
	for _, option := range options {
		switch option := option.(type) {
		case jstore.IdOption:
			result = result && (item.entity.ID == option.Value)
		case jstore.CompareOption:
			if option.Operation == "=" {
				result = result && (item.object[option.Property] == option.Value)
			} else {
				return false, errors.New("unsupported compare option: " + option.Operation)
			}
		default:
			return false, errors.Errorf("unsupported option: %+v", option)
		}
	}
	return result, nil
}

type MemoryStore struct {
	mutex sync.RWMutex

	storage map[string]map[string]map[string]storageItem
}

func NewMemoryStore(baseURL string, options ...jstore.StoreOption) (jstore.Store, error) {
	return &MemoryStore{storage: map[string]map[string]map[string]storageItem{}}, nil
}

func (store *MemoryStore) Delete(id jstore.EntityID) error {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	if _, ok := store.storage[id.Project]; !ok {
		return nil
	}
	if _, ok := store.storage[id.Project][id.DocumentType]; !ok {
		return nil
	}

	item, ok := store.storage[id.Project][id.DocumentType][id.ID]
	if ok && (item.entity.Version != id.Version && id.Version != jstore.NoVersion) {
		return jstore.OptimisticLockingError
	}

	delete(store.storage[id.Project][id.DocumentType], id.ID)

	return nil
}

func (store *MemoryStore) Save(id jstore.EntityID, json string) (jstore.EntityID, error) {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	if _, ok := store.storage[id.Project]; !ok {
		store.storage[id.Project] = map[string]map[string]storageItem{}
	}
	if _, ok := store.storage[id.Project][id.DocumentType]; !ok {
		store.storage[id.Project][id.DocumentType] = map[string]storageItem{}
	}

	present, ok := store.storage[id.Project][id.DocumentType][id.ID]
	if ok && (present.entity.Version != id.Version && id.Version != jstore.NoVersion) {
		return present.entity.EntityID, jstore.OptimisticLockingError
	}

	entity := jstore.Entity{jstore.NewIDWithVersion(id.Project, id.DocumentType, id.ID, present.entity.Version+1), nil, json}
	item, err := newItem(entity)
	if err != nil {
		return jstore.EntityID{}, err
	}

	store.storage[id.Project][id.DocumentType][id.ID] = item

	return item.entity.EntityID, nil
}

func (store *MemoryStore) Get(id jstore.EntityID) (jstore.Entity, error) {
	return store.Find(id.Project, id.DocumentType, jstore.Id(id.ID))
}

func (store *MemoryStore) Find(project, documentType string, options ...jstore.Option) (jstore.Entity, error) {
	values, err := store.FindN(project, documentType, 1, options...)
	if err != nil {
		return jstore.Entity{}, err
	}
	if len(values) == 0 {
		return jstore.Entity{}, jstore.NotFound
	}
	return values[0], nil
}

func (store *MemoryStore) FindN(project, documentType string, maxCount int, options ...jstore.Option) ([]jstore.Entity, error) {
	store.mutex.RLock()
	defer store.mutex.RUnlock()

	if _, ok := store.storage[project]; !ok {
		return []jstore.Entity{}, jstore.NotFound
	}
	if _, ok := store.storage[project][documentType]; !ok {
		return []jstore.Entity{}, jstore.NotFound
	}

	list := store.storage[project][documentType]

	result := []jstore.Entity{}
	for _, item := range list {
		matches, err := item.matches(options...)
		if err != nil {
			return []jstore.Entity{}, err
		}
		if matches {
			result = append(result, item.entity)
		}

		if len(result) == maxCount {
			break
		}
	}

	return result, nil
}

func (store *MemoryStore) HealthCheck() error {
	return nil
}
