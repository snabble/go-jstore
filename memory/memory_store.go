package memory

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"encoding/json"

	"github.com/snabble/go-jstore/v2"
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
		return item, fmt.Errorf("could not unmarshall json: %w", err)
	}
	return item, nil
}

func (item *storageItem) matches(options ...jstore.Option) (bool, error) {
	result := true
	for _, option := range options {
		switch option := option.(type) {
		case jstore.SortOption:
			continue
		case jstore.IdOption:
			result = result && (item.entity.ID == option.Value)
		case jstore.CompareOption:
			switch option.Value.(type) {
			case string:
				value, ok := item.object[option.Property].(string)
				if !ok {
					return false, fmt.Errorf("should be string")
				}

				switch option.Operation {
				case "=":
					result = result && (value == option.Value)
				default:
					return false, fmt.Errorf("unsupported compare option: %s", option.Operation)
				}
			case time.Time:
				value, ok := item.object[option.Property].(string)
				if !ok {
					return false, fmt.Errorf("should be string")
				}

				t, err := time.Parse("2006-01-02T15:04:05Z", value)
				if err != nil {
					return false, fmt.Errorf("not a date: %w", err)
				}

				s := option.Value.(time.Time)

				switch option.Operation {
				case "=":
					result = result && (t == s)
				case "<":
					result = result && (t.Before(s))
				case "<=":
					result = result && (t.Before(s) || t == s)
				case ">":
					result = result && (s.Before(t))
				case ">=":
					result = result && (s.Before(t) || s == t)
				default:
					return false, fmt.Errorf("unsupported compare option: %s", option.Operation)
				}

			case int:
				t, ok := item.object[option.Property].(float64)
				if !ok {
					return false, fmt.Errorf("not a number")
				}
				s := option.Value.(int)
				r, err := handleNumber(option.Operation, t, float64(s))
				if err != nil {
					return false, err
				}
				result = result && r

			case float64:
				t, ok := item.object[option.Property].(float64)
				if !ok {
					return false, fmt.Errorf("not a number")
				}
				s := option.Value.(float64)
				r, err := handleNumber(option.Operation, t, s)
				if err != nil {
					return false, err
				}
				result = result && r

			case int64:
				t, ok := item.object[option.Property].(float64)
				if !ok {
					return false, fmt.Errorf("not a number")
				}
				s := option.Value.(int64)
				r, err := handleNumber(option.Operation, t, float64(s))
				if err != nil {
					return false, err
				}
				result = result && r

			default:
				return false, fmt.Errorf("unsupported type for comparison")
			}

		default:
			return false, fmt.Errorf("unsupported option: %+v", option)
		}
	}
	return result, nil
}

func handleNumber(operation string, t, s float64) (bool, error) {
	switch operation {
	case "=":
		return (t == s), nil
	case "<":
		return (t < s), nil
	case "<=":
		return (t <= s), nil
	case ">":
		return (t > s), nil
	case ">=":
		return (t >= s), nil
	default:
		return false, fmt.Errorf("unsupported compare option: %s", operation)
	}
}

type MemoryStore struct {
	mutex sync.RWMutex

	storage map[string]map[string]map[string]storageItem
}

func NewMemoryStore(baseURL string, options ...jstore.StoreOption) (jstore.Store, error) {
	return &MemoryStore{storage: map[string]map[string]map[string]storageItem{}}, nil
}

type Version int

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
	if ok && (item.entity.Version != id.Version && id.Version != nil) {
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
	prevVersion, _ := id.Version.(Version)

	entity := jstore.Entity{
		EntityID: jstore.NewIDWithVersion(
			id.Project,
			id.DocumentType,
			id.ID,
			prevVersion+1,
		),
		ObjectRef: nil,
		JSON:      json,
	}
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

	items := []storageItem{}
	for _, item := range list {
		matches, err := item.matches(options...)
		if err != nil {
			return []jstore.Entity{}, err
		}
		if matches {
			items = append(items, item)
		}
	}

	if len(items) == 0 {
		return []jstore.Entity{}, nil
	}

	for _, o := range options {
		switch o.(type) {
		case jstore.SortOption:
			var err error
			items, err = store.sort(items, o.(jstore.SortOption))
			if err != nil {
				return []jstore.Entity{}, err
			}
		default:
		}
	}

	if len(items) > maxCount {
		items = items[:maxCount]
	}

	result := make([]jstore.Entity, 0, len(items))
	for _, item := range items {
		result = append(result, item.entity)
	}

	return result, nil
}

func (store *MemoryStore) sort(items []storageItem, option jstore.SortOption) ([]storageItem, error) {
	switch t := items[0].object[option.Property].(type) {
	case string:
		sort.Slice(items, func(i, j int) bool {
			return option.Ascending == (items[i].object[option.Property].(string) <= items[j].object[option.Property].(string))
		})
	case int:
		items = intSort(items, option.Property, option.Ascending)
	case int8:
		items = intSort(items, option.Property, option.Ascending)
	case int16:
		items = intSort(items, option.Property, option.Ascending)
	case int32:
		items = intSort(items, option.Property, option.Ascending)
	case int64:
		items = intSort(items, option.Property, option.Ascending)
	case float32:
		items = floatSort(items, option.Property, option.Ascending)
	case float64:
		items = floatSort(items, option.Property, option.Ascending)
	case time.Time:
		sort.Slice(items, func(i, j int) bool {
			return option.Ascending ==
				(items[i].object[option.Property].(time.Time).
					Before(items[j].object[option.Property].(time.Time)))
		})
	default:
		return []storageItem{}, fmt.Errorf("unsupported sort option on %s with type %T", option.Property, t)
	}
	return items, nil
}

func floatSort(items []storageItem, property string, ascending bool) []storageItem {
	sort.Slice(items, func(i, j int) bool {
		return ascending == (items[i].object[property].(float64) <= items[j].object[property].(float64))
	})
	return items
}

func intSort(items []storageItem, property string, ascending bool) []storageItem {
	sort.Slice(items, func(i, j int) bool {
		return ascending == (items[i].object[property].(int64) <= items[j].object[property].(int64))
	})
	return items
}

func (store *MemoryStore) HealthCheck() error {
	return nil
}
