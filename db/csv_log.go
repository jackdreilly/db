package db

import (
	"io"
	"encoding/csv"
)

func CreateCsvLogger(w io.WriteCloser) chan<-[]string {
	c := make(chan []string)
	cW := csv.NewWriter(w)
	go func() {
		defer w.Close()
		for r := range c {
			cW.Write(r)
			cW.Flush()
		}
	}()
	return c
}