package elastic

import (
	"encoding/json"
	"github.com/olivere/elastic"
	. "github.com/snabble/go-jstore"
	. "github.com/stretchr/testify/assert"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strconv"
	"syscall"
	"testing"
	"time"
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
	NoError(t, err)
	NoError(t, validStore.HealthCheck())
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
	NoError(t, err)

	invalidStore := &ElasticStore{
		client: esClient,
	}

	Error(t, invalidStore.HealthCheck())
}

func Test_BasicStoring(t *testing.T) {
	project := randStringBytes(10)
	personBucket, err := NewBucket("elastic", esTestURL(),
		project, "person", SyncUpdates)
	NoError(t, err)

	NoError(t, personBucket.Marshal(ford, "ford"))
	NoError(t, personBucket.Marshal(zaphod, "zaphod"))
	NoError(t, personBucket.Marshal(zaphod, "foo"))

	spaceshipBucket, err := NewBucket("elastic", esTestURL(),
		project, "spaceship", SyncUpdates)
	NoError(t, err)

	NoError(t, spaceshipBucket.Marshal(heartOfGold, "heartOfGold"))
	NoError(t, spaceshipBucket.Marshal(heartOfGold, "foo"))

	var result Person

	// find one person by id
	err = personBucket.Unmarshal(&result, Id("ford"))
	NoError(t, err)
	Equal(t, ford, result)

	// find one person by id
	err = personBucket.Unmarshal(&result, Id("foo"))
	NoError(t, err)
	Equal(t, zaphod, result)
}

func Test_FindInMissingProject(t *testing.T) {
	b, err := NewBucket("elastic", esTestURL(),
		randStringBytes(10), "person", SyncUpdates)
	NoError(t, err)

	// find one person by id
	_, err = b.Find(Id("ford"))
	Equal(t, NotFound, err)
}

func Test_CompatreOptions(t *testing.T) {
	b, err := NewBucket("elastic", esTestURL(),
		randStringBytes(10), "person", SyncUpdates)
	NoError(t, err)

	NoError(t, b.Marshal(ford, "ford"))
	NoError(t, b.Marshal(marvin, "marvin"))
	NoError(t, b.Marshal(zaphod, "zaphod"))

	tests := []struct {
		name     string
		options  []Option
		expected *Person
	}{
		{
			"integer equal",
			[]Option{Eq("age", 42)},
			&ford,
		},
		{
			"string equal",
			[]Option{Eq("name", "Ford Prefect")},
			&ford,
		},
		{
			" > ",
			[]Option{Gt("age", 42)},
			&zaphod,
		},
		{
			" > ",
			[]Option{Gt("age", 4200)},
			nil,
		},
		{
			" >= ",
			[]Option{Gte("age", 4200)},
			&zaphod,
		},
		{
			" < ",
			[]Option{Lt("age", 42)},
			nil,
		},
		{
			" <= ",
			[]Option{Lte("age", 42)},
			&ford,
		},
		{
			" gt and lt ",
			[]Option{Gt("age", 42), Lt("age", 4200)},
			&marvin,
		},
		{
			"date =",
			[]Option{Eq("birthDay", day("2042-01-01"))},
			&marvin,
		},
		{
			" < on date",
			[]Option{Lt("birthDay", day("1980-01-01"))},
			&zaphod,
		},
		{
			" <= and >= on date",
			[]Option{Lte("birthDay", day("1980-01-01")), Gte("birthDay", day("1980-01-01"))},
			&ford,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			result := &Person{}
			err = b.Unmarshal(result, test.options...)
			if test.expected == nil {
				Equal(t, NotFound, err)
			} else {
				NoError(t, err)
				Equal(t, test.expected, result)
			}
		})
	}
}

func Test_FindN(t *testing.T) {
	b, err := NewBucket("elastic", esTestURL(),
		randStringBytes(10), "person", SyncUpdates)
	NoError(t, err)

	for i := 0; i < 50; i++ {
		p := Person{
			Name: "person-" + strconv.Itoa(i),
			Age:  i,
		}
		err := b.Marshal(p, strconv.Itoa(i))
		NoError(t, err)
	}

	// find a subset
	docs, err := b.FindN(20)
	NoError(t, err)
	Equal(t, 20, len(docs))

	for _, d := range docs {
		p := Person{}
		err = json.Unmarshal([]byte(d), &p)
		NoError(t, err)
		Contains(t, p.Name, "person-")
	}

	// find all
	docs, err = b.FindN(1000)
	NoError(t, err)
	Equal(t, 50, len(docs))
}

func Test_Delete(t *testing.T) {
	b, err := NewBucket("elastic", esTestURL(),
		randStringBytes(10), "person", SyncUpdates)
	NoError(t, err)

	NoError(t, b.Marshal(ford, "ford"))
	NoError(t, b.Marshal(zaphod, "zaphod"))

	// ford is there
	var result Person
	NoError(t, b.Unmarshal(&result, Id("ford")))

	// delete ford
	NoError(t, b.Delete("ford"))

	// fort is away
	err = b.Unmarshal(&result, Id("ford"))
	Equal(t, NotFound, err)

	// but zaphod is still there
	NoError(t, b.Unmarshal(&result, Id("zaphod")))
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
