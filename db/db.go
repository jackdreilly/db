package db

import (
	"encoding/csv"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
)

const (
	defaultPort     = 8088
	defaultFilename = "/tmp/db.csv"
)

type Db struct {
	l          net.Listener
	d          map[string]string
	logger     chan<-[]string
}

type DbOptions struct {
	Filename  string
	Port      int32
	Overwrite bool
}

type ClientOptions struct {
	Port int32
}

func DefaultDbOptions() DbOptions {
	return DbOptions{defaultFilename, defaultPort, false}
}

func DefaultClientOptions() ClientOptions {
	return ClientOptions{defaultPort}
}

func (db *Db) Close() {
	db.l.Close()
}

func NewDb(o DbOptions) (*Db, error) {
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
		records, e := r.ReadAll()
		if e != nil {
			return db, e
		}
		for _, record := range records {
			if len(record) < 1 {
				return db, errors.New("Db log file should have at least 1 element")
			}
			if record[0] == "set" {
				if len(record) < 3 {
					return db, errors.New(fmt.Sprintf("Db log set command should have format 'set key value', was %v", record))
				}
				db.d[record[1]] = record[2]
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
	db.logger = CreateCsvLogger(logFile)
	connChan := SocketChannels(db.l)
	// Not sure how to not get away with this wait group.
	// We need to know when all connections are closed before closing the logger, because a connection may request
	// to dump something to the logger.
	wg := sync.WaitGroup{}
	go func() {
		defer func() {
			wg.Wait()
			close(db.logger)
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
						v, e := db.Get(r[1])
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
						e := db.Set(r[1], r[2])
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

func (db *Db) Get(key string) (string, error) {
	v, p := db.d[key]
	if !p {
		db.logM("keymiss", key, "")
		return v, errors.New("Could not find key " + key)
	}
	db.logM("get", key, v)
	return v, nil

}

func (db *Db) Set(key, value string) error {
	db.d[key] = value
	db.logM("set", key, value)
	return nil
}
func (db *Db) logM(r ...string) {
	db.logger <- r
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

func (c *Client) Get(key string) (string, error) {
	writer := csv.NewWriter(c.conn)
	e := writer.Write([]string{"get", key})
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

func (c *Client) Set(key, value string) error {
	writer := csv.NewWriter(c.conn)
	e := writer.Write([]string{"set", key, value})
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

func (client *Client) Close() {
	client.conn.Close()
}
