package db

import (
	"bytes"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestCommandFail(t *testing.T) {
	_, e := parseCommand([]string{})
	assert.NotNil(t, e)
	_, e = parseCommand([]string{""})
	assert.NotNil(t, e)
	_, e = parseCommand([]string{"net"})
	assert.NotNil(t, e)
	_, e = parseCommand([]string{"get"})
	assert.NotNil(t, e)
	_, e = parseCommand([]string{"set", "a"})
	assert.NotNil(t, e)
	_, e = parseCommand([]string{"set"})
	assert.NotNil(t, e)
	_, e = parseCommand([]string{"net", "key"})
	assert.NotNil(t, e)
	_, e = parseCommand([]string{"get", "+", "+", "a"})
	assert.NotNil(t, e)
	_, e = parseCommand([]string{"get", "a", "+", "+"})
	assert.NotNil(t, e)
	_, e = parseCommand([]string{"set", "a", "+", "+"})
	assert.NotNil(t, e)
	_, e = parseCommand([]string{"set", "a", "+", "+", "+", "+"})
	assert.NotNil(t, e)
}

func TestGetSimple(t *testing.T) {
	c, e := parseCommand([]string{"get", "tkey"})
	assert.Nil(t, e)
	assert.Equal(t, c, &command{
		ct:      command_get,
		top_key: "tkey",
		pos:     []commandValue{},
	})
}

func TestSetSimple(t *testing.T) {
	c, e := parseCommand([]string{"set", "abc", "cba"})
	assert.Nil(t, e)
	assert.Equal(t, c, &command{
		ct:        command_set,
		top_key:   "abc",
		pos:       []commandValue{},
		set_value: "cba",
	})
}

func TestGetString(t *testing.T) {
	c, e := parseCommand([]string{"get", "tkey", "_"})
	assert.Nil(t, e)
	assert.Equal(t, c, &command{
		ct:      command_get,
		top_key: "tkey",
		pos: []commandValue{
			commandValue{
				vt: valueString,
			},
		},
	})
}

func TestSetString(t *testing.T) {
	c, e := parseCommand([]string{"set", "tkey", "_", "val"})
	assert.Nil(t, e)
	assert.Equal(t, c, &command{
		ct:      command_set,
		top_key: "tkey",
		pos: []commandValue{
			commandValue{
				vt: valueString,
			},
		},
		set_value: "val",
	})
}

func TestSetStringNoUnderscoreSame(t *testing.T) {
	c, e := parseCommand([]string{"set", "tkey", "val"})
	assert.Nil(t, e)
	assert.Equal(t, c, &command{
		ct:        command_set,
		top_key:   "tkey",
		set_value: "val",
		pos:       []commandValue{},
	})
}

func TestGetWithSpecialCharacter(t *testing.T) {
	c, e := parseCommand([]string{"get", "+"})
	assert.Nil(t, e)
	assert.Equal(t, c, &command{
		ct:      command_get,
		top_key: "+",
		pos:     []commandValue{},
	})
}

func TestGetList(t *testing.T) {
	c, e := parseCommand([]string{"get", "key", "+"})
	assert.Nil(t, e)
	assert.Equal(t, c, &command{
		ct:      command_get,
		top_key: "key",
		pos: []commandValue{
			commandValue{
				vt: valueList,
				lc: listCommand{append: false, index: -1},
			},
		},
	})
}

func TestGetListIndex(t *testing.T) {
	c, e := parseCommand([]string{"get", "key", "+", "12"})
	assert.Nil(t, e)
	assert.Equal(t, c, &command{
		ct:      command_get,
		top_key: "key",
		pos: []commandValue{
			commandValue{
				vt: valueList,
				lc: listCommand{index: 12},
			},
		},
	})
}

func TestSetListAppend(t *testing.T) {
	c, e := parseCommand([]string{"set", "key", "+", "+", "V"})
	assert.Nil(t, e)
	assert.Equal(t, c, &command{
		ct:      command_set,
		top_key: "key",
		pos: []commandValue{
			commandValue{
				vt: valueList,
				lc: listCommand{append: true},
			},
		},
		set_value: "V",
	})
}

func TestSetListIndex(t *testing.T) {
	c, e := parseCommand([]string{"set", "key", "+", "31", "V"})
	assert.Nil(t, e)
	assert.Equal(t, c, &command{
		ct:      command_set,
		top_key: "key",
		pos: []commandValue{
			commandValue{
				vt: valueList,
				lc: listCommand{append: false, index: 31},
			},
		},
		set_value: "V",
	})
}

func TestSetListMap(t *testing.T) {
	c, e := parseCommand([]string{"set", "key", "+", "+", "->", "mykey", "abc"})
	assert.Nil(t, e)
	assert.Equal(t, c, &command{
		ct:      command_set,
		top_key: "key",
		pos: []commandValue{
			commandValue{
				vt: valueList,
				lc: listCommand{append: true},
			},
			commandValue{
				vt:  valueMap,
				key: "mykey",
			},
		},
		set_value: "abc",
	})
}

func TestGetMap(t *testing.T) {
	c, e := parseCommand([]string{"get", "key", "->", "inkey"})
	assert.Nil(t, e)
	assert.Equal(t, c, &command{
		ct:      command_get,
		top_key: "key",
		pos: []commandValue{
			commandValue{
				vt:  valueMap,
				key: "inkey",
			},
		},
	})
}

func TestHandleGet(t *testing.T) {
	x := storeValue{V: "a"}
	var b bytes.Buffer
	assert.Nil(t, json.NewEncoder(&b).Encode(x))
	c := command{
		ct: command_get,
		pos: []commandValue{
			commandValue{
				vt: valueString,
			},
		},
	}
	v, e := handleGet(b.String(), &c)
	assert.Nil(t, e)
	assert.Equal(t, "a", v)
}

func TestHandleGetSingleValue(t *testing.T) {
	x := storeValue{M: map[string]storeValue{"mykey": storeValue{V: "thevalue"}}}
	var b bytes.Buffer
	assert.Nil(t, json.NewEncoder(&b).Encode(x))
	c := command{
		ct: command_get,
		pos: []commandValue{
			commandValue{
				vt:  valueMap,
				key: "mykey",
			},
		},
	}
	v, e := handleGet(b.String(), &c)
	assert.Nil(t, e)
	assert.Equal(t, "thevalue", v)
}

func TestHandleGetMultiValue(t *testing.T) {
	x := storeValue{M: map[string]storeValue{
		"mykey": storeValue{V: "thevalue", L: []storeValue{storeValue{V: "hi"}}}},
	}
	var b bytes.Buffer
	assert.Nil(t, json.NewEncoder(&b).Encode(x))
	c := command{
		ct: command_get,
		pos: []commandValue{
			commandValue{
				vt:  valueMap,
				key: "mykey",
			},
		},
	}
	v, e := handleGet(b.String(), &c)
	assert.Nil(t, e)
	expected := storeValue{
		V: "thevalue",
		L: []storeValue{
			storeValue{
				V: "hi",
			},
		},
	}
	var actual storeValue
	assert.Nil(t, json.NewDecoder(strings.NewReader(v)).Decode(&actual))
	assert.Equal(t, actual, expected)
}

func TestHandleGetMultiValueStringValue(t *testing.T) {
	x := storeValue{M: map[string]storeValue{
		"mykey": storeValue{V: "thevalue", L: []storeValue{storeValue{V: "hi"}}}},
	}
	var b bytes.Buffer
	assert.Nil(t, json.NewEncoder(&b).Encode(x))
	c := command{
		ct: command_get,
		pos: []commandValue{
			commandValue{
				vt:  valueMap,
				key: "mykey",
			},
			commandValue{
				vt: valueString,
			},
		},
	}
	v, e := handleGet(b.String(), &c)
	assert.Nil(t, e)
	assert.Equal(t, "thevalue", v)
}

func TestHandleGetList(t *testing.T) {
	x := storeValue{
		L: []storeValue{
			storeValue{
				V: "value a",
			},
			storeValue{
				V: "value b",
			},
		},
	}
	var b bytes.Buffer
	assert.Nil(t, json.NewEncoder(&b).Encode(x))
	c := command{
		ct: command_get,
		pos: []commandValue{
			commandValue{
				vt: valueList,
				lc: listCommand{
					index: 1,
				},
			},
		},
	}
	v, e := handleGet(b.String(), &c)
	assert.Nil(t, e)
	assert.Equal(t, "value b", v)
}

func TestHandleSet(t *testing.T) {
	x := storeValue{
		V: "a",
	}
	expected := storeValue{
		V: "b",
	}
	var b bytes.Buffer
	assert.Nil(t, json.NewEncoder(&b).Encode(x))
	c := command{
		ct:        command_set,
		set_value: "b",
	}
	v, e := handleSet(b.String(), &c)
	assert.Nil(t, e)
	var actual storeValue
	assert.Nil(t, json.NewDecoder(strings.NewReader(v)).Decode(&actual))
	assert.Equal(t, expected, actual)
}

func TestHandleSetAppend(t *testing.T) {
	x := storeValue{}
	var b bytes.Buffer
	assert.Nil(t, json.NewEncoder(&b).Encode(x))
	c := command{
		ct: command_set,
		pos: []commandValue{
			commandValue{
				vt: valueList,
				lc: listCommand{
					append: true,
				},
			},
			commandValue{
				vt: valueList,
				lc: listCommand{
					append: true,
				},
			},
			commandValue{
				vt:  valueMap,
				key: "mykey",
			},
		},
		set_value: "b",
	}
	v, e := handleSet(b.String(), &c)
	assert.Nil(t, e)
	var actual storeValue
	assert.Nil(t, json.NewDecoder(strings.NewReader(v)).Decode(&actual))
	expected := storeValue{
		L: []storeValue{
			storeValue{
				L: []storeValue{
					storeValue{
						M: map[string]storeValue{
							"mykey": storeValue{
								V: "b",
							},
						},
					},
				},
			},
		},
	}
	assert.Equal(t, expected, actual)
}

func TestHandleSetGet(t *testing.T) {
	x := storeValue{}
	var b bytes.Buffer
	assert.Nil(t, json.NewEncoder(&b).Encode(x))
	c := command{
		ct: command_set,
		pos: []commandValue{
			commandValue{
				vt: valueList,
				lc: listCommand{
					append: true,
				},
			},
			commandValue{
				vt: valueList,
				lc: listCommand{
					append: true,
				},
			},
			commandValue{
				vt:  valueMap,
				key: "mykey",
			},
		},
		set_value: "b",
	}
	cGet := command{
		ct: command_get,
		pos: []commandValue{
			commandValue{
				vt: valueList,
				lc: listCommand{
					index: 0,
				},
			},
			commandValue{
				vt: valueList,
				lc: listCommand{
					index: 0,
				},
			},
			commandValue{
				vt:  valueMap,
				key: "mykey",
			},
		},
	}
	v, e := handleSet(b.String(), &c)
	assert.Nil(t, e)
	v, e = handleGet(v, &cGet)
	assert.Nil(t, e)
	assert.Equal(t, "b", v)
}

func TestHandleSetStringValue(t *testing.T) {
	x := storeValue{
		V: "a",
		L: []storeValue{
			storeValue{
				V: "b",
			},
		},
	}
	var b bytes.Buffer
	assert.Nil(t, json.NewEncoder(&b).Encode(x))
	c := command{
		ct: command_set,
		pos: []commandValue{
			commandValue{
				vt: valueString,
			},
		},
		set_value: "c",
	}
	v, e := handleSet(b.String(), &c)
	assert.Nil(t, e)
	var actual storeValue
	assert.Nil(t, json.NewDecoder(strings.NewReader(v)).Decode(&actual))
	expected := storeValue{
		V: "c",
		L: []storeValue{
			storeValue{
				V: "b",
			},
		},
	}
	assert.Equal(t, expected, actual)
}

func TestHandleSetStringImplicit(t *testing.T) {
	x := storeValue{
		V: "a",
		L: []storeValue{
			storeValue{
				V: "b",
			},
		},
	}
	var b bytes.Buffer
	assert.Nil(t, json.NewEncoder(&b).Encode(x))
	c := command{
		ct:        command_set,
		set_value: "c",
	}
	v, e := handleSet(b.String(), &c)
	assert.Nil(t, e)
	var actual storeValue
	assert.Nil(t, json.NewDecoder(strings.NewReader(v)).Decode(&actual))
	expected := storeValue{
		V: "c",
		L: []storeValue{
			storeValue{
				V: "b",
			},
		},
	}
	assert.Equal(t, expected, actual)
}
