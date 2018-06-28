package elastic

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/olivere/elastic"

	"github.com/snabble/go-jstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func esTestURL() string {
	if url, ok := syscall.Getenv("ES_TEST_URL"); ok {
		return url
	}
	return "http://127.0.0.1:9200"
}

type Person struct {
	Name     string    `json:"name"`
	Age      int       `json:"age"`
	BirthDay time.Time `json:"birthDay"`
}

type Spaceship struct {
	Name  string `json:"name"`
	Speed int    `json:"speed"`
}

var (
	ford        = Person{"Ford Prefect", 42, day("1980-01-01")}
	marvin      = Person{"Marvin", 1010, day("2042-01-01")}
	zaphod      = Person{"Zaphod Beeblebrox", 4200, day("1900-01-01")}
	heartOfGold = Spaceship{"Heart Of Gold", 99999999}
)

func Test_Health_OK(t *testing.T) {
	validStore, err := NewElasticStore(esTestURL())
	assert.NoError(t, err)
	assert.NoError(t, validStore.HealthCheck())
}

func Test_Health_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status" : "red"}`))
	}))
	defer server.Close()

	esClient, err := elastic.NewClient(
		elastic.SetURL(server.URL),
		elastic.SetHealthcheck(false),
		elastic.SetSniff(false))
	assert.NoError(t, err)

	invalidStore := &ElasticStore{
		client: esClient,
	}

	assert.Error(t, invalidStore.HealthCheck())
}

func Test_BasicStoring(t *testing.T) {
	project := randStringBytes(10)
	personBucket, err := jstore.NewBucket("elastic", esTestURL(),
		project, "person", SyncUpdates())
	assert.NoError(t, err)

	_, err = personBucket.Marshal(ford, jstore.NewID(project, "persons", "ford"))
	assert.NoError(t, err)
	_, err = personBucket.Marshal(zaphod, jstore.NewID(project, "persons", "zaphod"))
	assert.NoError(t, err)
	_, err = personBucket.Marshal(zaphod, jstore.NewID(project, "persons", "foo"))
	assert.NoError(t, err)

	spaceshipBucket, err := jstore.NewBucket("elastic", esTestURL(),
		project, "spaceship", SyncUpdates())
	assert.NoError(t, err)

	_, err = spaceshipBucket.Marshal(heartOfGold, jstore.NewID(project, "persons", "heartOfGold"))
	assert.NoError(t, err)
	_, err = spaceshipBucket.Marshal(heartOfGold, jstore.NewID(project, "persons", "foo"))
	assert.NoError(t, err)

	var result Person

	// find one person by id
	err = personBucket.Unmarshal(&result, jstore.Id("ford"))
	assert.NoError(t, err)
	assert.Equal(t, ford, result)

	// find one person by id
	err = personBucket.Unmarshal(&result, jstore.Id("foo"))
	assert.NoError(t, err)
	assert.Equal(t, zaphod, result)
}

func Test_TimeBasedIndex(t *testing.T) {
	now := "2018.06.01"
	project := randStringBytes(10)
	personBucket, err := jstore.NewBucket(
		"elastic",
		esTestURL(),
		project,
		"person",
		SyncUpdates(),
		IndexName(func(project, documentType string, matchAll bool) string {
			prefix := fmt.Sprintf("test-%s-%s-", project, documentType)
			suffix := now
			if matchAll {
				suffix = "*"
			}
			return prefix + suffix
		}),
	)
	assert.NoError(t, err)

	_, err = personBucket.Marshal(ford, jstore.NewID(project, "persons", "ford"))
	assert.NoError(t, err)
	now = "2018.06.02"
	_, err = personBucket.Marshal(zaphod, jstore.NewID(project, "persons", "zaphod"))
	assert.NoError(t, err)
	now = "2018.06.03"
	_, err = personBucket.Marshal(zaphod, jstore.NewID(project, "persons", "foo"))
	assert.NoError(t, err)

	var result Person

	// find one person by id
	err = personBucket.Unmarshal(&result, jstore.Id("ford"))
	require.NoError(t, err)
	assert.Equal(t, ford, result)

	// find one person by id
	err = personBucket.Unmarshal(&result, jstore.Id("foo"))
	require.NoError(t, err)
	assert.Equal(t, zaphod, result)

	// find all
	list, err := personBucket.FindN(10)
	require.NoError(t, err)
	assert.Equal(t, 3, len(list))

	// delete
	err = personBucket.Delete(jstore.NewID(project, "persons", "foo"))
	require.NoError(t, err)

	err = personBucket.Unmarshal(&result, jstore.Id("foo"))
	require.Error(t, err)
}

func Test_OptimisticLocking_Update(t *testing.T) {
	project := randStringBytes(10)
	store, _ := jstore.NewStore("elastic", esTestURL(), SyncUpdates())

	id, _ := store.Marshal(ford, jstore.NewID(project, "person", "ford"))

	assert.Equal(t, int64(1), id.Version)

	updatedID, _ := store.Marshal(Person{"Ford Prefect", 43, day("1980-01-01")}, id)

	assert.Equal(t, int64(2), updatedID.Version)

	_, err := store.Marshal(Person{"Ford Prefect", 41, day("1980-01-01")}, id)

	assert.Equal(t, jstore.OptimisticLockingError, err)
}

func Test_OptimisticLocking_Delete(t *testing.T) {
	project := randStringBytes(10)
	store, _ := jstore.NewStore("elastic", esTestURL(), SyncUpdates())

	id, _ := store.Marshal(ford, jstore.NewID(project, "person", "ford"))
	store.Marshal(Person{"Ford Prefect", 43, day("1980-01-01")}, id)

	err := store.Delete(id)

	assert.Equal(t, jstore.OptimisticLockingError, err)
}

func Test_FindInMissingProject(t *testing.T) {
	b, err := jstore.NewBucket("elastic", esTestURL(), randStringBytes(10), "person", SyncUpdates())
	assert.NoError(t, err)

	// find one person by id
	_, err = b.Find(jstore.Id("ford"))
	assert.Equal(t, jstore.NotFound, err)
}

func Test_CompareOptions(t *testing.T) {
	project := randStringBytes(10)
	b, err := jstore.NewBucket("elastic", esTestURL(), project, "person", SyncUpdates())
	assert.NoError(t, err)

	_, err = b.Marshal(ford, jstore.NewID(project, "persons", "ford"))
	assert.NoError(t, err)
	_, err = b.Marshal(marvin, jstore.NewID(project, "persons", "marvin"))
	assert.NoError(t, err)
	_, err = b.Marshal(zaphod, jstore.NewID(project, "persons", "zaphod"))
	assert.NoError(t, err)

	tests := []struct {
		name     string
		options  []jstore.Option
		expected *Person
	}{
		{
			"integer equal",
			[]jstore.Option{jstore.Eq("age", 42)},
			&ford,
		},
		{
			"string equal",
			[]jstore.Option{jstore.Eq("name", "Ford Prefect")},
			&ford,
		},
		{
			" > ",
			[]jstore.Option{jstore.Gt("age", 42)},
			&zaphod,
		},
		{
			" > ",
			[]jstore.Option{jstore.Gt("age", 4200)},
			nil,
		},
		{
			" >= ",
			[]jstore.Option{jstore.Gte("age", 4200)},
			&zaphod,
		},
		{
			" < ",
			[]jstore.Option{jstore.Lt("age", 42)},
			nil,
		},
		{
			" <= ",
			[]jstore.Option{jstore.Lte("age", 42)},
			&ford,
		},
		{
			" gt and lt ",
			[]jstore.Option{jstore.Gt("age", 42), jstore.Lt("age", 4200)},
			&marvin,
		},
		{
			"date =",
			[]jstore.Option{jstore.Eq("birthDay", day("2042-01-01"))},
			&marvin,
		},
		{
			" < on date",
			[]jstore.Option{jstore.Lt("birthDay", day("1980-01-01"))},
			&zaphod,
		},
		{
			" <= and >= on date",
			[]jstore.Option{jstore.Lte("birthDay", day("1980-01-01")), jstore.Gte("birthDay", day("1980-01-01"))},
			&ford,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			result := &Person{}
			err = b.Unmarshal(&result, test.options...)
			if test.expected == nil {
				assert.Equal(t, jstore.NotFound, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expected, result)
			}
		})
	}
}

func Test_FindN(t *testing.T) {
	project := randStringBytes(10)
	b, err := jstore.NewBucket("elastic", esTestURL(), project, "person", SyncUpdates())
	assert.NoError(t, err)

	for i := 0; i < 50; i++ {
		p := Person{
			Name: "person-" + strconv.Itoa(i),
			Age:  i,
		}
		_, err := b.Marshal(p, jstore.NewID(project, "person", strconv.Itoa(i)))
		assert.NoError(t, err)
	}

	// find a subset
	docs, err := b.FindN(20)
	assert.NoError(t, err)
	assert.Equal(t, 20, len(docs))

	for _, d := range docs {
		p := Person{}
		err = json.Unmarshal([]byte(d.JSON), &p)
		assert.NoError(t, err)
		assert.Contains(t, p.Name, "person-")
	}

	// find all
	docs, err = b.FindN(1000)
	assert.NoError(t, err)
	assert.Equal(t, 50, len(docs))
}

func Test_Delete(t *testing.T) {
	project := randStringBytes(10)
	b, err := jstore.NewBucket("elastic", esTestURL(), project, "person", SyncUpdates())
	assert.NoError(t, err)

	_, err = b.Marshal(ford, jstore.NewID(project, "person", "ford"))
	assert.NoError(t, err)
	_, err = b.Marshal(zaphod, jstore.NewID(project, "person", "zaphod"))
	assert.NoError(t, err)

	// ford is there
	var result Person
	assert.NoError(t, b.Unmarshal(&result, jstore.Id("ford")))

	// delete ford
	assert.NoError(t, b.Delete(jstore.NewID(project, "person", "ford")))

	// fort is away
	err = b.Unmarshal(&result, jstore.Id("ford"))
	assert.Equal(t, jstore.NotFound, err)

	// but zaphod is still there
	assert.NoError(t, b.Unmarshal(&result, jstore.Id("zaphod")))
}

func Test_CustomSearch(t *testing.T) {
	project := randStringBytes(10)
	esStore, err := NewElasticStore(esTestURL(), SyncUpdates())
	require.NoError(t, err)
	store := jstore.WrapStore(esStore)

	_, err = store.Marshal(ford, jstore.NewID(project, "person", "ford"))
	assert.NoError(t, err)
	_, err = store.Marshal(marvin, jstore.NewID(project, "person", "marvin"))
	assert.NoError(t, err)

	search := esStore.SearchIn(project, "person")
	require.NotNil(t, search)

	query := elastic.NewBoolQuery()
	query.Must(elastic.NewTermQuery("name.keyword", "Marvin"))
	search.Query(query)

	resp, err := search.Do(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(1), resp.Hits.TotalHits)
}

const letterBytes = "abcdefghijklmnopqrstuvwxyz"

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

func randStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func day(theDay string) time.Time {
	dayPattern := "2006-01-02"
	t, err := time.Parse(dayPattern, theDay)
	if err != nil {
		panic(err)
	}
	return t
}
