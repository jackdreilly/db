package db

import (
	"bytes"
	"encoding/csv"
	"github.com/stretchr/testify/assert"
	"io"
	"testing"
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
	logger, done := CreateCsvLogger(w)
	logger <- []string{"howdy", "jack"}
	close(logger)
	<-done
	reader := csv.NewReader(w)
	r, e := reader.Read()
	assert.Nil(t, e)
	assert.Equal(t, []string{"howdy", "jack"}, r)
}
