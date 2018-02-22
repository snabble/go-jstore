package elastic

import (
	"context"
	"errors"
	es "github.com/olivere/elastic"
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

func (store *ElasticStore) Delete(project, documentType, id string) error {
	_, err := store.client.Delete().
		Refresh("true").
		Index(project).
		Type(documentType).
		Id(id).
		Do(store.cntx())
	return err
}

func (store *ElasticStore) Save(project, documentType, id string, json string) error {
	_, err := store.client.Index().
		Index(project).
		Refresh("true").
		Type(documentType).
		Id(id).
		BodyString(json).
		Do(store.cntx())
	return err
}

func (store *ElasticStore) Find(project, documentType string, matcher ...jstore.Matcher) (string, error) {
	resp, err := store.createSearch(project, documentType, matcher...).Do(store.cntx())
	if err != nil {
		return "", err
	}

	if resp.TotalHits() <= 0 {
		return "", jstore.NotFound
	}

	return string(*resp.Hits.Hits[0].Source), nil
}

func (store *ElasticStore) FindN(project, documentType string, maxCount int, matcher ...jstore.Matcher) ([]string, error) {
	return []string{}, errors.New("method FindN not implemented")
}

func (store *ElasticStore) cntx() context.Context {
	return context.Background()
}

func (store *ElasticStore) createSearch(project, documentType string, matcher ...jstore.Matcher) *es.SearchService {
	boolQuery := es.NewBoolQuery()
	for _, m := range matcher {
		switch m := m.(type) {
		case jstore.IdMatcher:
			boolQuery.Must(es.NewIdsQuery().Ids(m.Value))
		case jstore.EqMatcher:
			if _, isString := m.Value.(string); isString {
				boolQuery.Must(es.NewTermQuery(m.Property+".keyword", m.Value))
			} else {
				boolQuery.Must(es.NewTermQuery(m.Property, m.Value))
			}
		}
	}

	return store.client.Search(project).
		Type(documentType).
		Index(project).
		Query(boolQuery)
}
