package db

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"encoding/json"
	"strings"
)

const (
	defaultPort     = 8088
	defaultFilename = "/tmp/db.csv"
)

type Db struct {
	l      net.Listener
	d      map[string]string
	logger chan<- []string
}

type Options struct {
	Filename  string
	Port      int32
	Overwrite bool
}

type ClientOptions struct {
	Port int32
}

func DefaultDbOptions() Options {
	return Options{defaultFilename, defaultPort, false}
}

func DefaultClientOptions() ClientOptions {
	return ClientOptions{defaultPort}
}

func (db *Db) Close() {
	db.l.Close()
}

func NewDb(o Options) (*Db, error) {
	db := &Db{}
	db.d = map[string]string{}
	if o.Overwrite {
		os.Remove(o.Filename)
	}
	_, err := os.Stat(o.Filename)
	fileExists := !os.IsNotExist(err)
	if fileExists {
		f, e := os.Open(o.Filename)
		defer f.Close()
		if e != nil {
			return db, e
		}
		r := csv.NewReader(f)
		var records = make([][]string, 0)
		for {
			r.FieldsPerRecord = 0
			rec, e := r.Read()
			if e == io.EOF {
				break
			}
			records = append(records, rec)
		}
		if e != nil {
			return db, e
		}
		for _, record := range records {
			if len(record) < 1 {
				return db, errors.New("db log file should have at least 1 element")
			}
			if record[0] == "set" {
				if e = db.Set(record[1:]...); e != nil {
					return nil, e
				}
			}
		}
	}
	db.l, err = net.Listen("tcp", fmt.Sprintf(":%d", o.Port))
	if err != nil {
		return db, err
	}
	var logFile *os.File
	if fileExists {
		logFile, err = os.OpenFile(o.Filename, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	} else {
		logFile, err = os.Create(o.Filename)
	}
	if err != nil {
		panic(err)
	}
	var done <-chan bool
	db.logger, done = CreateCsvLogger(logFile)
	connChan := SocketChannels(db.l)
	// Not sure how to not get away with this wait group.
	// We need to know when all connections are closed before closing the logger, because a connection may request
	// to dump something to the logger.
	wg := sync.WaitGroup{}
	go func() {
		defer func() {
			wg.Wait()
			close(db.logger)
			<-done
		}()
		for c := range connChan {
			wg.Add(1)
			go func(c net.Conn) {
				defer wg.Done()
				defer c.Close()
				for {
					reader := csv.NewReader(c)
					writer := csv.NewWriter(c)
					r, e := reader.Read()
					if e != nil {
						db.logM("error", "read_request_csv_parse", e.Error())
						writer.Write([]string{"error", e.Error()})
						writer.Flush()
						return
					}
					if r[0] == "get" {
						v, e := db.Get(r[1:]...)
						if e != nil {
							writer.Write([]string{"error", e.Error()})
							writer.Flush()
							continue
						}
						writer.Write([]string{"ok", v})
						writer.Flush()
						continue
					} else if r[0] == "set" {
						if len(r) < 3 {
							writer.Write([]string{"error", fmt.Sprintf("set command requires 2 arguments, saw %v", r)})
							writer.Flush()
							continue
						}
						e := db.Set(r[1:]...)
						if e != nil {
							writer.Write([]string{"error", e.Error()})
							writer.Flush()
							continue
						}
						writer.Write([]string{"ok"})
						writer.Flush()
						continue
					} else {
						db.logM("error", "bad_command", r[0])
						writer.Write([]string{"error", "bad_command", r[0]})
						writer.Flush()
						return
					}
				}
			}(c)
		}
	}()
	return db, nil
}

func (db *Db) Get(r ...string) (string, error) {
	gr := []string{"get"}
	gr = append(gr, r...)
	c, e := parseCommand(gr)
	if e != nil {
		db.logM("errorget", e.Error())
		return "", e
	}
	existing, ok := db.d[c.top_key]
	if !ok {
		e = errors.New("top-level key miss " + c.top_key)
		db.logM("errorget", e.Error())
		return "", e
	}
	v, e := handleGet(existing, c)
	if e != nil {
		db.logM("errorget", e.Error())
		return "", e
	}
	db.logM("get", r...)
	return v, nil
}

func (db *Db) Set(r ...string) error {
	gr := []string{"set"}
	gr = append(gr, r...)
	c, e := parseCommand(gr)
	if e != nil {
		db.logM("errorset", e.Error())
		return e
	}
	v, e := handleSet(db.d[c.top_key], c)
	if e != nil {
		db.logM("errorget", e.Error())
		return e
	}
	db.d[c.top_key] = v
	db.logM("set", r...)
	return nil
}

func (db *Db) logM(s string, r ...string) {
	if db.logger == nil {
		return
	}
	c := []string{s}
	c = append(c, r...)
	db.logger <- c
}

type Client struct {
	conn net.Conn
}

func NewClient(o ClientOptions) (*Client, error) {
	c := &Client{}
	var err error
	c.conn, err = net.Dial("tcp", fmt.Sprintf(":%d", o.Port))
	return c, err
}

func (c *Client) Get(command ...string) (string, error) {
	writer := csv.NewWriter(c.conn)
	rs := []string{"get"}
	rs = append(rs, command...)
	e := writer.Write(rs)
	if e != nil {
		return "", e
	}
	writer.Flush()
	r, e := csv.NewReader(c.conn).Read()
	if e != nil {
		return "", e
	}
	if r[0] == "error" {
		return "", errors.New(r[1])
	}
	return r[1], nil
}

func (c *Client) Set(command ...string) error {
	writer := csv.NewWriter(c.conn)
	rs := []string{"set"}
	rs = append(rs, command...)
	e := writer.Write(rs)
	if e != nil {
		return e
	}
	writer.Flush()
	reader := csv.NewReader(c.conn)
	r, e := reader.Read()
	if e != nil {
		return e
	}
	if r[0] == "error" {
		return errors.New(r[1])
	}
	return nil
}

func (c *Client) GetList(key string) ([]string, error) {
	r, e := c.Get(key)
	if e != nil{
		return nil, e
	}
	var s storeValue
	if e = json.NewDecoder(strings.NewReader(r)).Decode(&s); e != nil {
		return []string{}, nil
	}
	if s.L == nil {
		return []string{}, nil
	}
	v := make([]string, len(s.L))
	for i := 0; i < len(v); i++ {
		v[i] = s.L[i].V
	}
	return v, nil
}

func (c *Client) Append(key, value string) error {
	return c.Set(key, "+","+",value)
}


func (client *Client) Close() {
	client.conn.Close()
}
