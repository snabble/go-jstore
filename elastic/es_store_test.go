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
	ford   = Person{"Ford Perfect", 42}
	zappod = Person{"Zaphod Beeblebrox", 4200}

	project      = "earth" // randStringBytes(10)
	documentType = "person"
)

func Test_ElasticStore(t *testing.T) {
	s, err := NewElasticStore(esTestURL())
	NoError(t, err)

	err = s.Marshal(ford, project, documentType, "ford")
	NoError(t, err)

	err = s.Marshal(zappod, project, documentType, "zaphod")
	NoError(t, err)

	// hack needed for synchronous tesing
	s.(*ElasticStore).flush(project)

	var result Person

	// find one person by id
	err = s.Unmarshal(&result, project, documentType, Id("ford"))
	NoError(t, err)
	Equal(t, ford, result)

	// find one person by attribute
	err = s.Unmarshal(&result, project, documentType, Eq("Age", 42))
	NoError(t, err)
	Equal(t, ford, result)

	err = s.Unmarshal(&result, project, documentType, Eq("Name", "Ford Perfect"))
	NoError(t, err)
	Equal(t, ford, result)

	// No matches
	err = s.Unmarshal(&result, project, documentType, Eq("Name", "ford"))
	Equal(t, err, NotFound)
}

const letterBytes = "abcdefghijklmnopqrstuvwxyz"

func randStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
