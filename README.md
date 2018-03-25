# db

[![Build Status](https://travis-ci.org/stretchr/testify.svg)](https://travis-ci.org/jackdreilly/db)

Super duper simple, csv-backed tcp key-value store

Database backing http://comments.reillybrothers.net and comments.js

## How to start server:

```
cd server
go run main.go --port 8089 --file /tmp/mydb.csv
```

## How to connect to server:

### Command Line

```
netcat localhost 8089
set,a,b
get,a
```

### Golang client

Library in db.go provides a Client type, which has `Get`, `Set`, `GetList`, and `Append` methods, which simplify direct TCP access.

## Lists and Maps via Extended Grammar

Simple access is permitted via:

### Get

```
get,my key
```

### Set

```
set,my key, my value
```

But the database supports list and map storage types as well. Note that all extended commands will always require a top-level key immediately after the command (e.g., get or set), which serves as a namespace for further extended commands. Below, we use the namespace "my top key" for examples.

### List Append

```
set,my top key,+,+,appended value
set,my top key,+,0,appended value
```

The double `+` means append the "appended value" to the list structure in "my top key"

Appending is not supported for "get" command, for obvious reasons.

### List Modify (or List Indexing)

```
set,my top key,+,0,changed value
get,my top key,+,0
```

The `+` with a following `index` means change the value at index `index` (the index is 0 here) to "changed value".

### Map Values

```
set,my top key,->,inner key,inner value
get,my top key,->,inner key
```

The `->` with a following `key` means change (or set) the keyed value located at key "inner key" to "inner value".

### Arbitrary chaining

```
set,my top key,->,inner key,->,another inner key,+,+,+,+,->,super inner key,super inner value
get,my top key,->,inner key,->,another inner key,+,0,+,0,->,super inner key
```

Any arbitrary combination of list and map commands may be chained together to create complex storage.

### Structured Values

```
set,my top key,->,inner key,->,another inner key,+,+,+,+,->,super inner key,super inner value
get,my top key,->,inner key,->,another inner key,+,0,+,0,->,super inner key
get,my top key,->,inner key,->,another inner key,+,0,+,0
get,my top key,->,inner key,->,another inner key,+,0
get,my top key,->,inner key,->,another inner key
get,my top key,->,inner key
get,my top key
```

If a "get" command is underspecified, like `get,my top key,->,inner key`, then the command will return a JSON-formatted value capturing all sub-structure underneath the underspecified command. The format is:

```
{
  "V": <stringValue>,
  "L": [
     {
       "V": <stringValue>,
       "L": [...
       "M": {
         "m-key": ...
       }
     }
   ],
   "M": {
     "m-key": {
       "V": <stringValue>,
       ...
     }
   }
}
```
       

