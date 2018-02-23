package elastic

import (
	"context"
	es "github.com/olivere/elastic"
	"github.com/pkg/errors"
	"github.com/snabble/go-jstore"
)

func init() {
	jstore.RegisterProvider("elastic", NewElasticStore)
}

type ElasticStore struct {
	client *es.Client
}

func NewElasticStore(baseURL string) (jstore.Store, error) {
	client, err := es.NewClient(es.SetURL(baseURL))
	return &ElasticStore{
		client: client,
	}, err
}

func (store *ElasticStore) Delete(project, documentType, id string, options ...jstore.Option) error {
	query := store.client.Delete().
		Index(project).
		Type(documentType).
		Id(id)

	if len(options) == 1 && options[0] == jstore.SyncUpdates {
		query = query.Refresh("true")
	}

	_, err := query.Do(store.cntx())
	return err
}

func (store *ElasticStore) Save(project, documentType, id string, json string, options ...jstore.Option) error {
	query := store.client.Index().
		Index(project).
		Type(documentType).
		Id(id).
		BodyString(json)

	if len(options) == 1 && options[0] == jstore.SyncUpdates {
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

	return store.client.Search(project).
		Type(documentType).
		Index(project).
		Query(boolQuery), nil
}
