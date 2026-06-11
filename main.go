package main

import (
	database "atomixDB/database"
	"fmt"
	"math/rand"
	"os"
)

func main() {
	//database.StoreImpl()

	db := database.DB{
		Path: "test.db",
	}

	if err := db.Open(); err != nil {
		panic(err)
	}
	defer db.Close()

	users := &database.TableDef{
		Name: "users",
		Cols: []string{"id", "name"},
		Types: []uint32{
			database.TYPE_INT64,
			database.TYPE_BYTES,
		},
		PKeys: 1,
	}

	err := db.TableNew(users)
	fmt.Println("Create Table:", err)

	rec := database.Record{}
	rec.AddInt64("id", 1)
	rec.AddStr("name", []byte("Rahul"))

	ok, err := db.Insert("users", rec)
	fmt.Println("Insert:", ok, err)

	search := database.Record{}
	search.AddInt64("id", 1)

	ok, err = db.Get("users", &search)
	fmt.Println("Get:", ok, err)

	if ok {
		fmt.Println("Name:", string(search.Get("name").Str))
	}

	// ---------------- UPDATE TEST ----------------

rec = database.Record{}
rec.AddInt64("id", 1)
rec.AddStr("name", []byte("Amit"))

ok, err = db.Update("users", rec)
fmt.Println("Update:", ok, err)

search = database.Record{}
search.AddInt64("id", 1)

ok, err = db.Get("users", &search)
fmt.Println("Get After Update:", ok, err)

if ok {
	fmt.Println("Name:", string(search.Get("name").Str))
}

// ---------------- DELETE TEST ----------------

del := database.Record{}
del.AddInt64("id", 1)

ok, err = db.Delete("users", del)
fmt.Println("Delete:", ok, err)

// Try to fetch the deleted record
search = database.Record{}
search.AddInt64("id", 1)

ok, err = db.Get("users", &search)
fmt.Println("Get After Delete:", ok, err)

if ok {
	fmt.Println("Name:", string(search.Get("name").Str))
} else {
	fmt.Println("Record successfully deleted")
}

fmt.Println("\n----- Insert Records For Scan -----")

records := []struct {
	id   int64
	name string
}{
	{1, "Rahul"},
	{2, "Amit"},
	{3, "John"},
	{4, "Alice"},
	{5, "Bob"},
}

for _, r := range records {
	rec := database.Record{}
	rec.AddInt64("id", r.id)
	rec.AddStr("name", []byte(r.name))

	ok, err := db.Insert("users", rec)
	fmt.Println("Insert:", r.id, ok, err)
}

fmt.Println("\n----- Scan id = 2 to id = 4 -----")

var sc database.Scanner

sc.Cmp1 = database.CMP_GE
sc.Cmp2 = database.CMP_LE

sc.Key1.AddInt64("id", 2)
sc.Key2.AddInt64("id", 4)

err = db.Scan("users", &sc)
if err != nil {
	panic(err)
}

for sc.Valid() {
	var rec database.Record

	sc.Deref(&rec)

	fmt.Printf(
		"id=%d, name=%s\n",
		rec.Get("id").I64,
		string(rec.Get("name").Str),
	)

	sc.Next()
}
}

// Issues -
// 1. Truncates file before updating it
// 2. Writing data to file may not be atomic
// 3. The data is probably still in the operating system’s page cache after the write syscall returns
func SaveData1(path string, data []byte) error {
	fp, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0664)
	if err != nil {
		return err
	}
	defer fp.Close()
	_, err = fp.Write(data)
	return err
}

// Issue - Doesnt control when the data is persisted to the disk & the metadata might
// be persisted to the disk before data causing the file could be corrupted when the system crashes
func SaveData2(path string, data []byte) error {
	tmp := fmt.Sprintf("%s.tmp.%d", path, rand.Int())
	fp, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0664)
	if err != nil {
		return err
	}
	defer fp.Close()
	_, err = fp.Write(data)
	if err != nil {
		/// Remove the tmp file if the operation failed
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

func SaveData3(path string, data []byte) error {
	tmp := fmt.Sprintf("%s.tmp.%d", path, rand.Int())
	fp, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0664)
	if err != nil {
		return err
	}
	defer fp.Close()
	_, err = fp.Write(data)
	if err != nil {
		/// Remove the tmp file if the operation failed
		os.Remove(tmp)
		return err
	}
	err = fp.Sync()
	if err != nil {
		/// Remove the tmp file if the operation failed
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}