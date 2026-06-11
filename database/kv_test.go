package database

import (
	"os"
	"testing"
	"bytes"
)

func TestTableOperations(t *testing.T) {
	testDBPath := "test_table.db"
	defer os.Remove(testDBPath)

	db := DB{
		Path: testDBPath,
	}

	if err := db.Open(); err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	users := &TableDef{
		Name: "users",
		Cols: []string{"id", "name"},
		Types: []uint32{
			TYPE_INT64,
			TYPE_BYTES,
		},
		PKeys: 1,
	}

	if err := db.TableNew(users); err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	t.Run("Insert and Get", func(t *testing.T) {
		rec := Record{}
		rec.AddInt64("id", 1)
		rec.AddStr("name", []byte("Rahul"))

		ok, err := db.Insert("users", rec)
		if err != nil || !ok {
			t.Fatalf("Insert failed: %v", err)
		}

		search := Record{}
		search.AddInt64("id", 1)

		ok, err = db.Get("users", &search)
		if err != nil || !ok {
			t.Fatalf("Get failed: %v", err)
		}

		if string(search.Get("name").Str) != "Rahul" {
			t.Fatalf("Expected Rahul, got %s",
				string(search.Get("name").Str))
		}
	})

	t.Run("Update", func(t *testing.T) {
		rec := Record{}
		rec.AddInt64("id", 1)
		rec.AddStr("name", []byte("Amit"))

		ok, err := db.Update("users", rec)
		if err != nil || !ok {
			t.Fatalf("Update failed: %v", err)
		}

		search := Record{}
		search.AddInt64("id", 1)

		ok, err = db.Get("users", &search)
		if err != nil || !ok {
			t.Fatalf("Get after update failed: %v", err)
		}

		if string(search.Get("name").Str) != "Amit" {
			t.Fatalf("Expected Amit, got %s",
				string(search.Get("name").Str))
		}
	})

	t.Run("Delete", func(t *testing.T) {
		rec := Record{}
		rec.AddInt64("id", 1)

		ok, err := db.Delete("users", rec)
		if err != nil || !ok {
			t.Fatalf("Delete failed: %v", err)
		}

		search := Record{}
		search.AddInt64("id", 1)

		ok, err = db.Get("users", &search)
		if err != nil {
			t.Fatal(err)
		}

		if ok {
			t.Fatal("Record should have been deleted")
		}
	})

	t.Run("Duplicate Insert", func(t *testing.T) {
		rec := Record{}
		rec.AddInt64("id", 2)
		rec.AddStr("name", []byte("Alice"))

		ok, err := db.Insert("users", rec)
		if err != nil || !ok {
			t.Fatal("First insert failed")
		}

		ok, err = db.Insert("users", rec)

		if ok {
			t.Fatal("Duplicate insert should fail")
		}

		if err == nil {
			t.Fatal("Expected duplicate key error")
		}
	})

	t.Run("Update Missing Record", func(t *testing.T) {
		rec := Record{}
		rec.AddInt64("id", 100)
		rec.AddStr("name", []byte("Nobody"))

		ok, err := db.Update("users", rec)

		if ok {
			t.Fatal("Update should fail")
		}

		if err == nil {
			t.Fatal("Expected error")
		}
	})

	t.Run("Scan", func(t *testing.T) {
		records := []struct {
			id   int64
			name string
		}{
			{3, "John"},
			{4, "Alice"},
			{5, "Bob"},
		}

		for _, r := range records {
			rec := Record{}
			rec.AddInt64("id", r.id)
			rec.AddStr("name", []byte(r.name))

			_, err := db.Upsert("users", rec)
			if err != nil {
				t.Fatal(err)
			}
		}

		var sc Scanner

		sc.Cmp1 = CMP_GE
		sc.Cmp2 = CMP_LE

		sc.Key1.AddInt64("id", 2)
		sc.Key2.AddInt64("id", 4)

		if err := db.Scan("users", &sc); err != nil {
			t.Fatal(err)
		}

		expected := []struct {
			id   int64
			name string
		}{
			{2, "Alice"},
			{3, "John"},
			{4, "Alice"},
		}

		i := 0

		for sc.Valid() {
			var rec Record

			sc.Deref(&rec)

			if rec.Get("id").I64 != expected[i].id {
				t.Fatalf("Expected id %d, got %d",
					expected[i].id,
					rec.Get("id").I64)
			}

			if string(rec.Get("name").Str) != expected[i].name {
				t.Fatalf("Expected %s, got %s",
					expected[i].name,
					string(rec.Get("name").Str))
			}

			i++
			sc.Next()
		}

		if i != len(expected) {
			t.Fatalf("Expected %d rows, got %d",
				len(expected), i)
		}
	})
}

	func TestEscapeUnescapeString(t *testing.T) {
	testCases := [][]byte{
		[]byte(""),
		[]byte("Rahul"),
		[]byte{0},
		[]byte{1},
		[]byte{0, 1},
		[]byte("abc\x00xyz"),
		[]byte("abc\x01xyz"),
		[]byte{0, 1, 2, 3, 4},
	}

	for _, tc := range testCases {
		escaped := escapeString(tc)
		unescaped := unescapeString(escaped)

		if !bytes.Equal(tc, unescaped) {
			t.Fatalf("expected %v, got %v", tc, unescaped)
		}
	}
 }



func TestEncodeDecodeValues(t *testing.T) {
	values := []Value{
		{Type: TYPE_INT64, I64: 12345},
		{Type: TYPE_BYTES, Str: []byte("Rahul\x00X")},
	}

	encoded := encodeValues(nil, values)
	t.Log("Encoded bytes:", encoded)

	decoded, err := decodeValues(encoded, []uint32{
		TYPE_INT64,
		TYPE_BYTES,
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Decoded:", decoded)
}

func TestEncodeDecodeKey(t *testing.T) {
	values := []Value{
		{
			Type: TYPE_INT64,
			I64:  100,
		},
	}

	key := encodeKey(nil, 10, values)

	decoded, err := decodeKey(key, []uint32{
		TYPE_INT64,
	})

	if err != nil {
		t.Fatalf("decode key failed: %v", err)
	}

	if len(decoded) != 1 {
		t.Fatalf("expected 1 value, got %d", len(decoded))
	}

	if decoded[0].I64 != 100 {
		t.Fatalf("expected 100, got %d", decoded[0].I64)
	}
 }