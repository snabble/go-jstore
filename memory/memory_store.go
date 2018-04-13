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
	id     string
	raw    string
	object map[string]interface{}
}

func newItem(id, raw string) (storageItem, error) {
	item := storageItem{
		id:  id,
		raw: raw,
	}
	err := json.Unmarshal([]byte(raw), &item.object)
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
			result = result && (item.id == option.Value)
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

func (store *MemoryStore) Delete(project, documentType, id string) error {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	if _, ok := store.storage[project]; !ok {
		return nil
	}
	if _, ok := store.storage[project][documentType]; !ok {
		return nil
	}

	delete(store.storage[project][documentType], id)

	return nil
}

func (store *MemoryStore) Save(project, documentType, id string, json string) error {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	if _, ok := store.storage[project]; !ok {
		store.storage[project] = map[string]map[string]storageItem{}
	}
	if _, ok := store.storage[project][documentType]; !ok {
		store.storage[project][documentType] = map[string]storageItem{}
	}

	item, err := newItem(id, json)
	if err != nil {
		return err
	}

	store.storage[project][documentType][id] = item

	return nil
}

func (store *MemoryStore) Find(project, documentType string, options ...jstore.Option) (string, error) {
	values, err := store.FindN(project, documentType, 1, options...)
	if err != nil {
		return "", err
	}
	if len(values) == 0 {
		return "", jstore.NotFound
	}
	return values[0], nil
}

func (store *MemoryStore) FindN(project, documentType string, maxCount int, options ...jstore.Option) ([]string, error) {
	store.mutex.RLock()
	defer store.mutex.RUnlock()

	if _, ok := store.storage[project]; !ok {
		return []string{}, jstore.NotFound
	}
	if _, ok := store.storage[project][documentType]; !ok {
		return []string{}, jstore.NotFound
	}

	list := store.storage[project][documentType]

	result := []string{}
	for _, item := range list {
		matches, err := item.matches(options...)
		if err != nil {
			return []string{}, err
		}
		if matches {
			result = append(result, item.raw)
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
