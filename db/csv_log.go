package db

import (
	"io"
	"encoding/csv"
)

func CreateCsvLogger(w io.WriteCloser) (chan<-[]string, <-chan bool) {
	c := make(chan []string)
	d := make(chan bool)
	cW := csv.NewWriter(w)

	go func() {
		defer w.Close()
		for r := range c {
			cW.Write(r)
			//time.Sleep(time.Second)
			cW.Flush()
		}
		d<-true
	}()
	return c, d
}