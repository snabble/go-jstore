package elastic

import (
	"context"
	"strings"
	"time"

	"github.com/olivere/elastic"
	"github.com/pkg/errors"
	"github.com/snabble/go-jstore"
)

var DriverName = "elastic"

func init() {
	provider := func(baseURL string, options ...jstore.StoreOption) (jstore.Store, error) {
		return NewElasticStore(baseURL, options...)
	}
	jstore.RegisterProvider("elastic", provider)
}

type ElasticStore struct {
	client      *elastic.Client
	syncUpdates bool
}

func NewElasticStore(baseURL string, options ...jstore.StoreOption) (*ElasticStore, error) {
	client, err := elastic.NewClient(elastic.SetURL(baseURL))

	return &ElasticStore{
		client:      client,
		syncUpdates: len(options) == 1 && options[0] == jstore.SyncUpdates,
	}, err
}

func (store *ElasticStore) HealthCheck() error {
	cntx, cancelFunc := context.WithTimeout(store.cntx(), time.Second)
	defer cancelFunc()
	resp, err := store.client.ClusterHealth().
		Do(cntx)
	if err != nil {
		return errors.Wrap(err, "elasticsearch health")
	}
	if resp.Status != "green" && resp.Status != "yellow" {
		return errors.Errorf("elasticsearch health status is %v", resp.Status)
	}
	return nil
}

func (store *ElasticStore) Delete(id jstore.EntityID) error {
	query := store.client.Delete().
		Index(indexName(id.Project, id.DocumentType)).
		Type(id.DocumentType).
		Id(id.ID)

	if id.Version != jstore.NoVersion {
		query.Version(id.Version)
	}

	if store.syncUpdates {
		query = query.Refresh("true")
	}

	_, err := query.Do(store.cntx())

	if err != nil {
		if e, ok := err.(*elastic.Error); ok && e.Details.Type == "version_conflict_engine_exception" {
			return jstore.OptimisticLockingError
		}
	}

	return err
}

func (store *ElasticStore) Save(id jstore.EntityID, json string) (jstore.EntityID, error) {
	query := store.client.Index().
		Index(indexName(id.Project, id.DocumentType)).
		Type(id.DocumentType).
		Id(id.ID).
		BodyString(json)

	if id.Version != jstore.NoVersion {
		query.Version(id.Version)
	}

	if store.syncUpdates {
		query = query.Refresh("true")
	}

	resp, err := query.Do(store.cntx())

	if err != nil {
		if e, ok := err.(*elastic.Error); ok && e.Details.Type == "version_conflict_engine_exception" {
			return jstore.EntityID{}, jstore.OptimisticLockingError
		}
		return id, err
	}

	return jstore.EntityID{
		Project:      id.Project,
		DocumentType: id.DocumentType,
		ID:           resp.Id,
		Version:      resp.Version,
	}, nil
}

func (store *ElasticStore) Get(id jstore.EntityID) (jstore.Entity, error) {
	return store.Find(id.Project, id.DocumentType, jstore.Id(id.ID))
}

func (store *ElasticStore) Find(project, documentType string, options ...jstore.Option) (jstore.Entity, error) {
	search, err := store.createSearch(project, documentType, options...)
	if err != nil {
		return jstore.Entity{}, err
	}

	resp, err := search.Size(1).Do(store.cntx())

	if err != nil {
		if e, ok := err.(*elastic.Error); ok &&
			(e.Details.Type == "index_not_found_exception" ||
				e.Details.Reason == "no such index") {
			return jstore.Entity{}, jstore.NotFound
		}
		return jstore.Entity{}, err
	}

	if resp.TotalHits() <= 0 {
		return jstore.Entity{}, jstore.NotFound
	}

	return toEntity(project, documentType, resp.Hits.Hits[0]), nil
}

func (store *ElasticStore) FindN(project, documentType string, maxCount int, options ...jstore.Option) ([]jstore.Entity, error) {
	search, err := store.createSearch(project, documentType, options...)
	if err != nil {
		return nil, err
	}

	resp, err := search.Size(maxCount).Do(store.cntx())
	if err != nil {
		if e, ok := err.(*elastic.Error); ok &&
			(e.Details.Type == "index_not_found_exception" ||
				e.Details.Reason == "no such index") {
			return nil, jstore.NotFound
		}
		return nil, err
	}

	results := make([]jstore.Entity, 0, resp.TotalHits())
	for _, h := range resp.Hits.Hits {
		results = append(results, toEntity(project, documentType, h))
	}
	return results, nil
}

func (store *ElasticStore) SearchIn(project, documentType string) *elastic.SearchService {
	return store.client.
		Search(indexName(project, documentType)).
		Type(documentType)
}

func (store *ElasticStore) cntx() context.Context {
	return context.Background()
}

func toEntity(project, documentType string, hit *elastic.SearchHit) jstore.Entity {
	return jstore.Entity{
		toEntityID(project, documentType, hit),
		nil,
		string(*hit.Source),
	}
}

func toEntityID(project, documentType string, hit *elastic.SearchHit) jstore.EntityID {
	return jstore.EntityID{
		Project:      project,
		DocumentType: documentType,
		ID:           hit.Id,
		Version:      *hit.Version,
	}
}

func (store *ElasticStore) createSearch(project, documentType string, options ...jstore.Option) (*elastic.SearchService, error) {
	boolQuery := elastic.NewBoolQuery()
	for _, o := range options {
		switch o := o.(type) {
		case jstore.IdOption:
			boolQuery.Must(elastic.NewIdsQuery().Ids(o.Value))
		case jstore.CompareOption:
			switch o.Operation {
			case "=":
				if _, isString := o.Value.(string); isString {
					boolQuery.Must(elastic.NewTermQuery(o.Property+".keyword", o.Value))
				} else {
					boolQuery.Must(elastic.NewTermQuery(o.Property, o.Value))
				}
			case "<":
				boolQuery.Must(elastic.NewRangeQuery(o.Property).Lt(o.Value))
			case "<=":
				boolQuery.Must(elastic.NewRangeQuery(o.Property).Lte(o.Value))
			case ">":
				boolQuery.Must(elastic.NewRangeQuery(o.Property).Gt(o.Value))
			case ">=":
				boolQuery.Must(elastic.NewRangeQuery(o.Property).Gte(o.Value))
			default:
				return nil, errors.New("unsupported compare option: " + o.Operation)
			}
		default:
			return nil, errors.Errorf("unsupported option: %v", o)
		}
	}

	return store.SearchIn(project, documentType).
		Version(true).
		Query(boolQuery), nil
}

func indexName(project, documentType string) string {
	return strings.ToLower(project + "-" + documentType)
}
