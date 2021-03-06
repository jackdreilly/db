package db

import (
	"encoding/csv"
	"fmt"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
)

func DbOptionsTest() Options {
	o := DefaultDbOptions()
	o.Filename = ".test.csv"
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


	assert.Nil(t, c.Set("mapkey", "->", "key", "value"))
	v, e = c.Get("mapkey", "->", "key")
	assert.Nil(t,e)
	assert.Equal(t, "value", v)

	assert.Nil(t, c.Append("ap", "apa"))
	assert.Nil(t, c.Append("ap", "apb"))
	l, e := c.GetList("ap")
	assert.Nil(t,e)
	assert.Equal(t,[]string{"apa", "apb"}, l)

	l, e = c.GetList("a")
	assert.Nil(t,e)
	assert.Empty(t,l)

	l, e = c.GetList("mapkey")
	assert.Nil(t,e)
	assert.Empty(t,l)
}

func TestTcp(t *testing.T) {
	o := DbOptionsTest()
	o.Port = 23421
	db, e := NewDb(o)
	defer db.Close()
	assert.Nil(t, e)
	c, e := net.Dial("tcp", fmt.Sprintf(":%d", o.Port))
	assert.Nil(t, e)
	writer := csv.NewWriter(c)
	reader := csv.NewReader(c)

	e = writer.Write([]string{"set", "a"})
	writer.Flush()
	assert.Nil(t, e)

	r, e := reader.Read()
	assert.Nil(t, e)
	assert.NotEmpty(t, r)
	assert.Equal(t, "error", r[0])

	writer = csv.NewWriter(c)
	reader = csv.NewReader(c)
	e = writer.Write([]string{"set", "a", "b"})
	writer.Flush()
	assert.Nil(t, e)

	r, e = reader.Read()
	assert.Nil(t, e)
	assert.NotEmpty(t, r)
	assert.Equal(t, "ok", r[0])
}


