package db

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"bytes"
	"encoding/csv"
	"fmt"
)

func DbOptionsTest() DbOptions {
	o := DefaultDbOptions()
	o.Overwrite = true
	return o
}

func TestDb(t *testing.T) {
	o := DbOptionsTest()
	db, e := NewDb(o)
	assert.Nil(t, e)
	_, e = db.Get("a")
	assert.NotNil(t, e)
	assert.Nil(t, db.Set("a", "b"))
	v, e := db.Get("a")
	assert.Nil(t, e)
	assert.Equal(t, v, "b")
	assert.Nil(t, db.Set("a", "c"))
	v, e = db.Get("a")
	assert.Nil(t, e)
	assert.Equal(t, v, "c")
	db.Close()
	o.Overwrite = false
	db, e = NewDb(o)
	defer db.Close()
	assert.Nil(t, e)
	v, e = db.Get("a")
	assert.Nil(t, e)
	assert.Equal(t, "c", v)
}

func TestCsv(t *testing.T) {
	b := &bytes.Buffer{}
	reader := csv.NewReader(b)
	fmt.Println(b.String())
	csv.NewWriter(b).Write([]string{"hi"})
	fmt.Println(b.String())
	assert.Equal(t, "hi", b.String())
	r, e := reader.Read()
	assert.Nil(t,e)
	assert.Equal(t, []string{"hi"}, r)

}

func TestClient(t *testing.T) {
	db, e := NewDb(DbOptionsTest())
	defer db.Close()
	assert.Nil(t, e)
	c, e := NewClient(DefaultClientOptions())
	defer c.Close()
	assert.Nil(t, e)
	_, e = c.Get("a")
	assert.NotNil(t, e)
	assert.Nil(t, c.Set("a", "b"))
	v, e := c.Get("a")
	assert.Nil(t, e)
	assert.Equal(t, "b", v)
}