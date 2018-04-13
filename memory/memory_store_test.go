package memory

import (
	"encoding/json"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/snabble/go-jstore"
	. "github.com/stretchr/testify/assert"
)

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

func Test_BasicStoring(t *testing.T) {
	project := randStringBytes(10)
	personBucket, err := jstore.NewBucket("memory", "memory", project, "person", jstore.SyncUpdates)
	NoError(t, err)

	NoError(t, personBucket.Marshal(ford, "ford"))
	NoError(t, personBucket.Marshal(zaphod, "zaphod"))
	NoError(t, personBucket.Marshal(zaphod, "foo"))

	spaceshipBucket, err := jstore.NewBucket("memory", "memory", project, "spaceship", jstore.SyncUpdates)
	NoError(t, err)

	NoError(t, spaceshipBucket.Marshal(heartOfGold, "heartOfGold"))
	NoError(t, spaceshipBucket.Marshal(heartOfGold, "foo"))

	var result Person

	// find one person by id
	err = personBucket.Unmarshal(&result, jstore.Id("ford"))
	NoError(t, err)
	Equal(t, ford, result)

	// find one person by id
	err = personBucket.Unmarshal(&result, jstore.Id("foo"))
	NoError(t, err)
	Equal(t, zaphod, result)
}

func Test_FindInMissingProject(t *testing.T) {
	b, err := jstore.NewBucket("memory", "memory", randStringBytes(10), "person", jstore.SyncUpdates)
	NoError(t, err)

	// find one person by id
	_, err = b.Find(jstore.Id("ford"))
	Equal(t, jstore.NotFound, err)
}

func Test_CompareOptions(t *testing.T) {
	b, err := jstore.NewBucket("memory", "memory", randStringBytes(10), "person", jstore.SyncUpdates)
	NoError(t, err)

	NoError(t, b.Marshal(ford, "ford"))
	NoError(t, b.Marshal(marvin, "marvin"))
	NoError(t, b.Marshal(zaphod, "zaphod"))

	result := Person{}
	err = b.Unmarshal(&result, jstore.Eq("name", "Ford Prefect"))
	NoError(t, err)
	Equal(t, ford, result)
}

func Test_FindN(t *testing.T) {
	b, err := jstore.NewBucket("memory", "memory", randStringBytes(10), "person", jstore.SyncUpdates)
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
	b, err := jstore.NewBucket("memory", "memory", randStringBytes(10), "person", jstore.SyncUpdates)
	NoError(t, err)

	NoError(t, b.Marshal(ford, "ford"))
	NoError(t, b.Marshal(zaphod, "zaphod"))

	// ford is there
	var result Person
	NoError(t, b.Unmarshal(&result, jstore.Id("ford")))

	// delete ford
	NoError(t, b.Delete("ford"))

	// fort is away
	err = b.Unmarshal(&result, jstore.Id("ford"))
	Equal(t, jstore.NotFound, err)

	// but zaphod is still there
	NoError(t, b.Unmarshal(&result, jstore.Id("zaphod")))
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
