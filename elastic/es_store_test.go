package elastic

import (
	. "github.com/snabble/go-jstore"
	. "github.com/stretchr/testify/assert"
	"math/rand"
	"syscall"
	"testing"
)

func esTestURL() string {
	if url, ok := syscall.Getenv("ES_TEST_URL"); ok {
		return url
	}
	return "http://127.0.0.1:9200"
}

type Person struct {
	Name string
	Age  int
}

var (
	ford   = Person{"Ford Prefect", 42}
	zaphod = Person{"Zaphod Beeblebrox", 4200}
)

func Test_ElasticStore(t *testing.T) {
	b, err := NewBucket("elastic", esTestURL(),
		randStringBytes(10), "person")
	NoError(t, err)

	NoError(t, b.Marshal(ford, "ford"))
	NoError(t, b.Marshal(zaphod, "zaphod"))

	var result Person

	// find one person by id
	err = b.Unmarshal(&result, Id("ford"))
	NoError(t, err)
	Equal(t, ford, result)

	// find one person by attribute
	err = b.Unmarshal(&result, Eq("Age", 42))
	NoError(t, err)
	Equal(t, ford, result)

	err = b.Unmarshal(&result, Eq("Name", "Ford Prefect"))
	NoError(t, err)
	Equal(t, ford, result)

	// No matches
	err = b.Unmarshal(&result, Eq("Name", "ford"))
	Equal(t, err, NotFound)
}

func Test_Delete(t *testing.T) {
	b, err := NewBucket("elastic", esTestURL(),
		randStringBytes(10), "person")
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

func randStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
