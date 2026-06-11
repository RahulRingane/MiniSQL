package database

func (db *DB) Open() error {
	db.kv.Path = db.Path
	db.tables = make(map[string]*TableDef)
	return db.kv.Open()
}

func (db *DB) Close() {
	db.kv.Close()
}