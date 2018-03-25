package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type commandType int
type commandValueType int

const (
	command_set commandType = iota
	command_get commandType = iota
)

const (
	valueString commandValueType = iota
	valueList   commandValueType = iota
	valueMap    commandValueType = iota
)

type storeValue struct {
	V string
	L []storeValue
	M map[string]storeValue
}

type listCommand struct {
	append bool
	index  int
}

type commandValue struct {
	vt  commandValueType
	key string
	lc  listCommand
}

type command struct {
	ct        commandType
	pos       []commandValue
	top_key   string
	set_value string
}

func handleGet(previous string, c *command) (string, error) {
	s := storeValue{}
	e := json.NewDecoder(strings.NewReader(previous)).Decode(&s)
	if e != nil {
		if len(c.pos) == 0 {
			return previous, nil
		}
		return "", e
	}
	for _, v := range c.pos {
		switch v.vt {
		case valueString:
			return s.V, nil
		case valueList:
			if v.lc.index < 0 {
				b, e := json.Marshal(s.L)
				if e != nil {
					return "", e
				}
				return string(b), nil
			}
			if v.lc.index < 0 || v.lc.index >= len(s.L) {
				return "", errors.New(fmt.Sprintf("index request out of range: %d vs %d", v.lc.index, len(s.L)))
			}
			s = s.L[v.lc.index]
			break
		case valueMap:
			s = s.M[v.key]
			break
		default:
			return "", errors.New("unexpected value type")
		}
	}
	if len(s.M) == 0 && len(s.L) == 0 {
		return s.V, nil
	}
	b, e := json.Marshal(s)
	if e != nil {
		return "", e
	}
	return string(b), nil
}

func changeMapValue(previous storeValue, pos []commandValue, v string) (storeValue, error) {
	if len(pos) == 0 {
		previous.V = v
		return previous, nil
	}
	p := pos[0]
	switch p.vt {
	case valueString:
		if len(pos[1:]) > 0 {
			return previous, errors.New("leftover positional arguments in set command")
		}
		previous.V = v
		return previous, nil
	case valueList:
		if p.lc.append {
			nv, e := changeNewValue(pos[1:], v)
			if e != nil {
				return previous, e
			}
			previous.L = append(previous.L, nv)
			return previous, nil
		}
		nv, e := changeMapValue(previous.L[p.lc.index], pos[1:], v)
		if e != nil {
			return previous, e
		}
		previous.L[p.lc.index] = nv
		return previous, nil
	case valueMap:
		nv, ok := previous.M[p.key]
		if ok {
			nv, e := changeMapValue(nv, pos[1:], v)
			if e != nil {
				return previous, e
			}
			previous.M[p.key] = nv
			return previous, nil
		}
		nv, e := changeNewValue(pos[1:], v)
		if e != nil {
			return previous, e
		}
		if previous.M == nil {
			previous.M = make(map[string]storeValue)
		}
		previous.M[p.key] = nv
		return previous, nil
	default:
		return previous, errors.New("do not understand set value type")
	}
}
func changeNewValue(pos []commandValue, s string) (storeValue, error) {
	return changeMapValue(storeValue{}, pos, s)
}

func handleSet(previous string, c *command) (string, error) {
	s := storeValue{}
	if len(previous) != 0 {
		e := json.NewDecoder(strings.NewReader(previous)).Decode(&s)
		if e != nil {
			if len(c.pos) == 0 {
				return c.set_value, nil
			}
			return "", e
		}
	}
	s, e := changeMapValue(s, c.pos, c.set_value)
	if e != nil {
		return "", e
	}
	b, e := json.Marshal(s)
	if e != nil {
		return "", e
	}
	return string(b), nil
}

func parseCommand(r []string) (*command, error) {
	c := &command{}
	c.pos = make([]commandValue, 0)
	if len(r) == 0 {
		return c, errors.New("empty request")
	}
	if len(r) < 2 {
		return c, errors.New("no top key provided")
	}
	c.top_key = r[1]
	switch r[0] {
	case "get":
		return parseGet(c, r[2:])
	case "set":
		return parseSet(c, r[2:])
	default:
		return c, errors.New(fmt.Sprintf("unknown command: %s", r[0]))
	}
}

func parseValue(c *command, r []string) (*command, error, []string) {
	if len(r) == 0 {
		return c, nil, r
	}
	switch r[0] {
	case "+":
		return parseListValue(c, r[1:])
	case "->":
		return parseMapValue(c, r[1:])
	case "_":
		return parseStringValue(c, r[1:])
	default:
		return c, errors.New("unexpected command " + r[0]), r[1:]
	}
}
func parseMapValue(c *command, r []string) (*command, error, []string) {
	v := commandValue{}
	v.key = r[0]
	v.vt = valueMap
	c.pos = append(c.pos, v)
	return parseValue(c, r[1:])
}
func parseListValue(c *command, r []string) (*command, error, []string) {
	v := commandValue{}
	v.vt = valueList
	if len(r) == 0 {
		if c.ct == command_get {
			v.lc.index = -1
			c.pos = append(c.pos, v)
			return c, nil, r
		}
		return c, errors.New("list command expects index/append value, none given"), r
	}

	var e error
	v.lc, e = parseListCommand(r[0])
	if v.lc.append && c.ct == command_get {
		return c, errors.New("no append command allowed in get calls"), r
	}
	if e != nil {
		return c, e, r
	}
	c.pos = append(c.pos, v)
	return parseValue(c, r[1:])
}
func parseListCommand(s string) (listCommand, error) {
	if s == "+" {
		return listCommand{append: true}, nil
	}
	i, e := strconv.Atoi(s)
	if e != nil {
		return listCommand{}, e
	}
	return listCommand{append: false, index: i}, nil
}
func parseStringValue(c *command, r []string) (*command, error, []string) {
	v := commandValue{}
	v.vt = valueString
	c.pos = append(c.pos, v)
	return c, nil, r
}

func parseSet(c *command, r []string) (*command, error) {
	c.ct = command_set
	if len(r) == 0 {
		return c, errors.New("No key or value command provided for set command")
	}
	c.set_value = r[len(r)-1]
	c, e, r := parseValue(c, r[:len(r)-1])
	if e != nil {
		return c, e
	}
	return c, e
}
func parseGet(c *command, r []string) (*command, error) {
	c.ct = command_get
	c, e, r := parseValue(c, r)
	if e != nil {
		return c, e
	}
	if len(r) != 0 {
		return c, errors.New(fmt.Sprintf("Extra values received on get: %V", r))
	}
	return c, e
}
