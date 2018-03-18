package db

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"bytes"
	"io"
	"encoding/csv"
)

type TestWriteCloser struct {
	bytes.Buffer
}

func (c *TestWriteCloser) Close() error {
	return nil
}

func TestCreateCsvLogger(t *testing.T) {
	var w io.ReadWriteCloser
	w = &TestWriteCloser{}
	logger := CreateCsvLogger(w)
	defer close(logger)
	logger<-[]string{"howdy","jack"}
	r, e := csv.NewReader(w).Read()
	assert.Nil(t,e)
	assert.Equal(t, []string{"howdy","jack"}, r)
}