package memory

import (
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/snabble/go-jstore/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Person struct {
	Name     string    `json:"name"`
	Age      int       `json:"age"`
	BirthDay time.Time `json:"birthDay"`
}

var (
	ford   = Person{"Ford Prefect", 42, day("1980-01-01")}
	marvin = Person{"Marvin", 1010, day("2042-01-01")}
	zaphod = Person{"Zaphod Beeblebrox", 4200, day("1900-01-01")}
)

func Test_BasicStoring(t *testing.T) {
	store, err := jstore.NewStore("memory", "memory")
	require.NoError(t, err)

	id, err := store.Marshal(ford, jstore.NewID("project", "person", "ford"))
	require.NoError(t, err)
	assert.Equal(t, jstore.EntityID{Project: "project", DocumentType: "person", ID: "ford", Version: Version(1)}, id)

	_, err = store.Marshal(zaphod, jstore.NewID("project", "person", "zaphod"))
	require.NoError(t, err)
	_, err = store.Marshal(zaphod, jstore.NewID("project", "person", "zaphod"))
	require.NoError(t, err)
	_, err = store.Marshal(zaphod, jstore.NewID("project", "person", "foo"))
	require.NoError(t, err)

	// find one person by id
	result := Person{}
	err = store.Unmarshal(&jstore.Entity{ObjectRef: &result}, "project", "person", jstore.Id("ford"))
	require.NoError(t, err)
	assert.Equal(t, ford, result)

	// find one person by id
	err = store.Unmarshal(&result, "project", "person", jstore.Id("foo"))
	require.NoError(t, err)
	assert.Equal(t, zaphod, result)
}

func Test_OptimisticLocking_Update(t *testing.T) {
	store, _ := jstore.NewStore("memory", "memory")

	id, _ := store.Marshal(ford, jstore.NewID("project", "person", "ford"))

	assert.Equal(t, Version(1), id.Version)

	updatedID, _ := store.Marshal(Person{"Ford Prefect", 43, day("1980-01-01")}, id)

	assert.Equal(t, Version(2), updatedID.Version)

	conflictedID, err := store.Marshal(Person{"Ford Prefect", 41, day("1980-01-01")}, id)

	assert.Equal(t, Version(2), conflictedID.Version)
	assert.Equal(t, jstore.OptimisticLockingError, err)
}

func Test_OptimisticLocking_Delete(t *testing.T) {
	store, _ := jstore.NewStore("memory", "memory")

	id, _ := store.Marshal(ford, jstore.NewID("project", "person", "ford"))
	store.Marshal(Person{"Ford Prefect", 43, day("1980-01-01")}, id)

	err := store.Delete(id)

	assert.Equal(t, jstore.OptimisticLockingError, err)
}

func Test_FindInMissingProject(t *testing.T) {
	store, _ := jstore.NewStore("memory", "memory")

	_, err := store.Find("project", "person", jstore.Id("ford"))

	assert.Equal(t, jstore.NotFound, err)
}

func Test_CompareOptions(t *testing.T) {
	store, err := jstore.NewStore("memory", "memory")
	require.NoError(t, err)

	store.Marshal(ford, jstore.NewID("project", "person", "ford"))
	store.Marshal(marvin, jstore.NewID("project", "person", "marvin"))
	store.Marshal(zaphod, jstore.NewID("project", "person", "zaphod"))

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
			[]jstore.Option{jstore.Gt("age", 1010)},
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
			err = store.Unmarshal(&result, "project", "person", test.options...)
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
	store, err := jstore.NewStore("memory", "memory")
	require.NoError(t, err)

	for i := 0; i < 50; i++ {
		p := Person{
			Name: "person-" + strconv.Itoa(i),
			Age:  i,
		}
		_, err := store.Marshal(p, jstore.NewID("project", "person", strconv.Itoa(i)))
		require.NoError(t, err)
	}

	// find a subset
	docs, err := store.FindN("project", "person", 20)
	require.NoError(t, err)
	assert.Equal(t, 20, len(docs))

	for _, d := range docs {
		p := Person{}
		err = json.Unmarshal([]byte(d.JSON), &p)
		require.NoError(t, err)
		assert.Contains(t, p.Name, "person-")
	}

	// find all
	docs, err = store.FindN("project", "person", 1000)
	require.NoError(t, err)
	assert.Equal(t, 50, len(docs))
}

func Test_FindN_SortBy(t *testing.T) {
	store, err := jstore.NewStore("memory", "memory")
	require.NoError(t, err)

	for i := 0; i < 50; i++ {
		p := Person{
			Name: "person-" + strconv.Itoa(i),
			Age:  i,
		}
		_, err := store.Marshal(p, jstore.NewID("project", "person", strconv.Itoa(i)))
		require.NoError(t, err)
	}

	// ascending
	docs, err := store.FindN("project", "person", 20, jstore.SortBy("age", true))
	require.NoError(t, err)
	assert.Equal(t, 20, len(docs))

	age := -1
	for _, d := range docs {
		p := Person{}
		err = json.Unmarshal([]byte(d.JSON), &p)
		require.NoError(t, err)
		assert.True(t, age <= p.Age)
		age = p.Age
	}

	// descending
	docs, err = store.FindN("project", "person", 20, jstore.SortBy("age", false))
	require.NoError(t, err)
	assert.Equal(t, 20, len(docs))

	age = 9999999
	for _, d := range docs {
		p := Person{}
		err = json.Unmarshal([]byte(d.JSON), &p)
		require.NoError(t, err)
		assert.True(t, age >= p.Age)
		age = p.Age
	}

}

func Test_Delete(t *testing.T) {
	store, err := jstore.NewStore("memory", "memory")
	require.NoError(t, err)

	store.Marshal(ford, jstore.NewID("project", "person", "ford"))
	store.Marshal(zaphod, jstore.NewID("project", "person", "zaphod"))

	// ford is there
	var result Person
	require.NoError(t, store.Unmarshal(&result, "project", "person", jstore.Id("ford")))

	// delete ford
	require.NoError(t, store.Delete(jstore.NewID("project", "person", "ford")))

	// ford is away
	err = store.Unmarshal(&result, "project", "person", jstore.Id("ford"))
	assert.Equal(t, jstore.NotFound, err)

	// but zaphod is still there
	require.NoError(t, store.Unmarshal(&result, "project", "person", jstore.Id("zaphod")))
}

func day(theDay string) time.Time {
	dayPattern := "2006-01-02"
	t, err := time.Parse(dayPattern, theDay)
	if err != nil {
		panic(err)
	}
	return t
}
