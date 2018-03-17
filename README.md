# db
Super duper simple, csv-backed tcp key-value store

How to start server:

cd server
go run main.go --port 8089 --file /tmp/mydb.csv


How to connect to server:

netcat localhost 8089
set,a,b
get,a
