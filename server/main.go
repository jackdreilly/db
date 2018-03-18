package main

import (
	"flag"
	"os"
	"strconv"

	"github.com/jackdreilly/db/db"
	"os/signal"
	"fmt"
)

var (
	filename = flag.String("file", "", "Optional path to db file")
	port = flag.Int64("port", 0, "TCP port to listen on, defaults to PORT env")
)

func main() {
	flag.Parse()
	o := db.DefaultDbOptions()
	if *port == 0 {
		p, f := os.LookupEnv("PORT")
		if f {
			var err error
			*port, err = strconv.ParseInt(p, 10, 64)
			check(err)
		}
	}
	if *port != 0 {
		o.Port = int32(*port)
	}
	if len(*filename) > 0 {
		o.Filename = *filename
	}
	fmt.Printf("Saving to %s and listening on port %d...\n", o.Filename, o.Port)
	d, e := db.NewDb(o)
	defer d.Close()
	check(e)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
}
func check(e error) {
	if e != nil {
		panic(e)
	}
}