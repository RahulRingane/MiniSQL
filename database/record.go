package database

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
)

const (
	TYPE_ERROR = 0
	TYPE_INT64 = 1
	TYPE_BYTES = 2
)

// table cell
type Value struct {
	Type uint32
	I64  int64
	Str  []byte
}

// table row
type Record struct {
	Cols []string
	Vals []Value
}

type DB struct {
	Path   string
	kv     KV
	tables map[string]*TableDef // cached table definition
}

type TableDef struct {
	Name   string
	Types  []uint32 // column types
	Cols   []string // column names
	PKeys  int      // the first PKeys columns are primary keys
	Prefix uint32   // B+Tree prefix
}

// internal table: metadata
var TDEF_META = &TableDef{
	Prefix: 1,
	Name:   "@meta",
	Types:  []uint32{TYPE_BYTES, TYPE_BYTES},
	Cols:   []string{"key", "val"},
	PKeys:  1,
}

// internal table: table schemas
var TDEF_TABLE = &TableDef{
	Prefix: 2,
	Name:   "@table",
	Types:  []uint32{TYPE_BYTES, TYPE_BYTES},
	Cols:   []string{"name", "def"},
	PKeys:  1,
}

func (rec *Record) AddStr(key string, val []byte) *Record {
	rec.Cols = append(rec.Cols, key)
	rec.Vals = append(rec.Vals, Value{
		Type: TYPE_BYTES,
		Str:  val,
	})
	return rec
}

func (rec *Record) AddInt64(key string, val int64) *Record {
	rec.Cols = append(rec.Cols, key)
	rec.Vals = append(rec.Vals, Value{
		Type: TYPE_INT64,
		I64:  val,
	})
	return rec
}

func (rec *Record) Get(key string) *Value {
	for i, col := range rec.Cols {
		if col == key {
			return &rec.Vals[i]
		}
	}
	return nil
}

func (db *DB) Get(table string, rec *Record) (bool, error) {
	tdef := getTableDef(db, table)
	if tdef == nil {
		return false, fmt.Errorf("table not found: %s", table)
	}
	return dbGet(db, tdef, rec)
}

func getTableDef(db *DB, name string) *TableDef {
	tdef, ok := db.tables[name]
	if !ok {
		if db.tables == nil {
			db.tables = map[string]*TableDef{}
		}

		tdef = getTableDefDB(db, name)

		if tdef != nil {
			db.tables[name] = tdef
		}
	}

	return tdef
}

func getTableDefDB(db *DB, name string) *TableDef {
	rec := (&Record{}).AddStr("name", []byte(name))

	ok, err := dbGet(db, TDEF_TABLE, rec)
	if err != nil || !ok {
		return nil
	}

	tdef := &TableDef{}

	if err := json.Unmarshal(rec.Get("def").Str, tdef); err != nil {
		return nil
	}

	return tdef
}

// get row by primary key
func dbGet(db *DB, tdef *TableDef, rec *Record) (bool, error) {
	sc := Scanner{
		Cmp1: CMP_GE,
		Key1: *rec,
		Cmp2: CMP_LE,
		Key2: *rec,
	}

	if err := dbScan(db, tdef, &sc); err != nil {
		return false, err
	}

	if !sc.Valid() {
		return false, nil
	}

	sc.Deref(rec)
	return true, nil
}

func encodeKey(out []byte, prefix uint32, vals []Value) []byte {
	var buf [4]byte

	binary.BigEndian.PutUint32(buf[:], prefix)

	out = append(out, buf[:]...)
	out = encodeValues(out, vals)

	return out
}

func encodeValues(out []byte, vals []Value) []byte {
	for _, v := range vals {

		switch v.Type {

		case TYPE_INT64:
			var buf [8]byte

			u := uint64(v.I64) + (1 << 63)

			binary.BigEndian.PutUint64(buf[:], u)

			out = append(out, buf[:]...)

		case TYPE_BYTES:
			out = append(out, escapeString(v.Str)...)
			out = append(out, 0)

		default:
			panic("invalid type")
		}
	}

	return out
}

func decodeValues(in []byte, types []uint32) ([]Value, error) {
	values := make([]Value, 0, len(types))
	index := 0

	for _, typ := range types {
		switch typ {

		case TYPE_INT64:
			if index+8 > len(in) {
				return nil, fmt.Errorf("invalid int64 encoding")
			}

			u := binary.BigEndian.Uint64(in[index : index+8])

			values = append(values, Value{
				Type: TYPE_INT64,
				I64:  int64(u - (1 << 63)),
			})

			index += 8

		case TYPE_BYTES:
			start := index

			for index < len(in) {
				if in[index] == 0 {
					break
				}

				// Skip escaped byte (0x01 xx)
				if in[index] == 0x01 {
					if index+1 >= len(in) {
						return nil, fmt.Errorf("invalid escaped string")
					}
					index += 2
				} else {
					index++
				}
			}

			if index >= len(in) {
				return nil, fmt.Errorf("unterminated string")
			}

			values = append(values, Value{
				Type: TYPE_BYTES,
				Str:  unescapeString(in[start:index]),
			})

			index++ // Skip null terminator

		default:
			return nil, fmt.Errorf("unknown type %d", typ)
		}
	}

	if index != len(in) {
		return nil, fmt.Errorf("extra bytes at end of value")
	}

	return values, nil
}

func decodeKey(in []byte, types []uint32) ([]Value, error) {
	if len(in) < 4 {
		return nil, fmt.Errorf("invalid key")
	}

	// Skip 4-byte table prefix.
	return decodeValues(in[4:], types)
}

// Strings are encoded as null-terminated strings.
// Escape embedded '\0' and '\1' bytes.
func escapeString(in []byte) []byte {
	zeros := bytes.Count(in, []byte{0})
	ones := bytes.Count(in, []byte{1})

	if zeros+ones == 0 {
		return in
	}

	out := make([]byte, len(in)+zeros+ones)

	pos := 0
	for _, ch := range in {
		if ch <= 1 {
			out[pos] = 0x01
			out[pos+1] = ch + 1
			pos += 2
		} else {
			out[pos] = ch
			pos++
		}
	}

	return out
}

func unescapeString(in []byte) []byte {
	if len(in) == 0 {
		return in
	}

	escapeCount := 0

	for i := 0; i < len(in); i++ {
		if in[i] == 0x01 && i+1 < len(in) {
			escapeCount++
			i++
		}
	}

	if escapeCount == 0 {
		return in
	}

	out := make([]byte, len(in)-escapeCount)

	pos := 0

	for i := 0; i < len(in); i++ {
		if in[i] == 0x01 && i+1 < len(in) {
			out[pos] = in[i+1] - 1
			pos++
			i++
		} else {
			out[pos] = in[i]
			pos++
		}
	}

	return out
}

func checkRecord(tdef *TableDef, rec Record, n int) ([]Value, error) {
	orderedValues := make([]Value, len(tdef.Cols))

	var limit int

	switch n {
	case tdef.PKeys:
		limit = tdef.PKeys

	case len(tdef.Cols):
		limit = len(tdef.Cols)

	default:
		return nil, fmt.Errorf("invalid number of columns")
	}

	for i := 0; i < limit; i++ {
		idx := indexOf(rec.Cols, tdef.Cols[i])

		if idx == -1 {
			return nil, fmt.Errorf("missing column: %s", tdef.Cols[i])
		}

		if rec.Vals[idx].Type != tdef.Types[i] {
			return nil, fmt.Errorf(
				"type mismatch for column %s: expected %d, got %d",
				tdef.Cols[i],
				tdef.Types[i],
				rec.Vals[idx].Type,
			)
		}

		orderedValues[i] = rec.Vals[idx]
	}

	return orderedValues, nil
}

// Helper functions

func contains(slice []string, item string) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

func indexOf(slice []string, item string) int {
	for i, v := range slice {
		if v == item {
			return i
		}
	}
	return -1
}