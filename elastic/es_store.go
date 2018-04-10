package elastic

import (
	"context"
	es "github.com/olivere/elastic"
	"github.com/pkg/errors"
	"github.com/snabble/go-jstore"
	"strings"
	"time"
)

var DriverName = "elastic"

func init() {
	jstore.RegisterProvider("elastic", NewElasticStore)
}

type ElasticStore struct {
	client      *es.Client
	syncUpdates bool
}

func NewElasticStore(baseURL string, options ...jstore.StoreOption) (jstore.Store, error) {
	client, err := es.NewClient(es.SetURL(baseURL))
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

func (store *ElasticStore) Delete(project, documentType, id string) error {
	query := store.client.Delete().
		Index(indexName(project, documentType)).
		Type(documentType).
		Id(id)

	if store.syncUpdates {
		query = query.Refresh("true")
	}

	_, err := query.Do(store.cntx())
	return err
}

func (store *ElasticStore) Save(project, documentType, id string, json string) error {
	query := store.client.Index().
		Index(indexName(project, documentType)).
		Type(documentType).
		Id(id).
		BodyString(json)

	if store.syncUpdates {
		query = query.Refresh("true")
	}

	_, err := query.Do(store.cntx())
	return err
}

func (store *ElasticStore) Find(project, documentType string, options ...jstore.Option) (string, error) {
	search, err := store.createSearch(project, documentType, options...)
	if err != nil {
		return "", err
	}

	resp, err := search.Size(1).Do(store.cntx())

	if err != nil {
		if e, ok := err.(*es.Error); ok &&
			(e.Details.Type == "index_not_found_exception" ||
				e.Details.Reason == "no such index") {
			return "", jstore.NotFound
		}
		return "", err
	}

	if resp.TotalHits() <= 0 {
		return "", jstore.NotFound
	}

	return string(*resp.Hits.Hits[0].Source), nil
}

func (store *ElasticStore) FindN(project, documentType string, maxCount int, options ...jstore.Option) ([]string, error) {
	search, err := store.createSearch(project, documentType, options...)
	if err != nil {
		return nil, err
	}

	resp, err := search.Size(maxCount).Do(store.cntx())
	if err != nil {
		if e, ok := err.(*es.Error); ok &&
			(e.Details.Type == "index_not_found_exception" ||
				e.Details.Reason == "no such index") {
			return nil, jstore.NotFound
		}
		return nil, err
	}

	results := make([]string, 0, resp.TotalHits())
	for _, h := range resp.Hits.Hits {
		results = append(results, string(*h.Source))
	}
	return results, nil
}

func (store *ElasticStore) cntx() context.Context {
	return context.Background()
}

func (store *ElasticStore) createSearch(project, documentType string, options ...jstore.Option) (*es.SearchService, error) {
	boolQuery := es.NewBoolQuery()
	for _, o := range options {
		switch o := o.(type) {
		case jstore.IdOption:
			boolQuery.Must(es.NewIdsQuery().Ids(o.Value))
		case jstore.CompareOption:
			switch o.Operation {
			case "=":
				if _, isString := o.Value.(string); isString {
					boolQuery.Must(es.NewTermQuery(o.Property+".keyword", o.Value))
				} else {
					boolQuery.Must(es.NewTermQuery(o.Property, o.Value))
				}
			case "<":
				boolQuery.Must(es.NewRangeQuery(o.Property).Lt(o.Value))
			case "<=":
				boolQuery.Must(es.NewRangeQuery(o.Property).Lte(o.Value))
			case ">":
				boolQuery.Must(es.NewRangeQuery(o.Property).Gt(o.Value))
			case ">=":
				boolQuery.Must(es.NewRangeQuery(o.Property).Gte(o.Value))
			default:
				return nil, errors.New("unsupported compare option: " + o.Operation)
			}
		default:
			return nil, errors.Errorf("unsupported option: %v", o)
		}
	}

	return store.client.
		Search(indexName(project, documentType)).
		Type(documentType).
		Query(boolQuery), nil
}

func indexName(project, documentType string) string {
	return strings.ToLower(project + "-" + documentType)
}
