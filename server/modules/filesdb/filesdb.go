package filesdb

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"time"

	"github.com/akokshar/storage/server/modules"
	// need to call this explicitly so it registers db driver
	_ "github.com/mattn/go-sqlite3"
)

const (
	actionAdd = iota
	actionTrash
	actionErase
	actionMoveOut
	actionMoveIn
)

type filesDB struct {
	startTime int64
	dbFile    string
	database  *sql.DB
	rootID    int64
}

// NewFilesDB initializes a db instance
func NewFilesDB(dbFile string) modules.FilesDB {
	var err error
	var database *sql.DB
	database, err = sql.Open("sqlite3", dbFile)
	if err != nil {
		log.Fatal(err)
	}

	db := new(filesDB)
	db.startTime = time.Now().Unix()
	db.dbFile = dbFile
	db.database = database

	_, err = database.Exec(`
		PRAGMA foreign_keys = ON;
		CREATE TABLE IF NOT EXISTS files (
			id INTEGER PRIMARY KEY AUTOINCREMENT, /* need ids not tu be reused */
			parent_id INTEGER,
			scan_time DATETIME,

			size  INTEGER,
			mdate INTEGER, 
			cdate INTEGER, 
			name  TEXT,
			ctype TEXT,

			CONSTRAINT fk_parent
				FOREIGN KEY (parent_id) 
				REFERENCES files (id)
				ON DELETE CASCADE,

			CONSTRAINT k_filename
				UNIQUE (parent_id, name)
				ON CONFLICT ROLLBACK
		);

		CREATE TABLE IF NOT EXISTS changelog (
			id INTEGER PRIMARY KEY AUTOINCREMENT, /* to be used as a sync anchor */
			parent_id INTEGER,
			file_id INTEGER,
			action INTEGER,

			CONSTRAINT k_lastchange
				UNIQUE (parent_id, file_id)
				ON CONFLICT REPLACE
		);
	`)
	if err != nil {
		log.Fatal(err)
	}

	row := database.QueryRow(`SELECT id FROM files WHERE parent_id IS NULL`)
	if err := row.Scan(&db.rootID); err != nil {
		res, err := database.Exec(`INSERT INTO files (name) VALUES ("ROOT")`)
		if err != nil {
			log.Fatal(err)
		}
		db.rootID, err = res.LastInsertId()
		if err != nil {
			log.Fatal(err)
		}
	}

	return db
}

func (m *filesDB) dbRemoveFile(id int64) (err error) {
	tx, err := m.database.Begin()
	if err != nil {
		return
	}

	var parentID int64
	row := tx.QueryRow(`SELECT parent_id FROM files where id = ?`, id)
	if err = row.Scan(&parentID); err != nil {
		return
	}

	_, err = tx.Exec(
		"delete from files where id = $1",
		id)
	if err != nil {
		tx.Rollback()
		return
	}

	_, err = tx.Exec(
		"insert into changelog (parent_id, file_id, action) values ($1, $2, $3)",
		parentID, id, actionErase)
	if err != nil {
		tx.Rollback()
		return
	}

	tx.Commit()
	return
}

func (m *filesDB) dbCreateItemPlaceholder(parentID int64, name string) (id int64, err error) {
	tx, err := m.database.Begin()
	if err != nil {
		panic(fmt.Sprintf("Failed at CreateItemPlaceholder '%v'", err))
	}

	res, err := tx.Exec("insert into files (parent_id, name) values ($1, $2)", parentID, name)
	if err != nil {
		tx.Rollback()
		return
	}

	id, err = res.LastInsertId()
	if err != nil {
		tx.Rollback()
		panic(fmt.Sprintf("Failed at CreateItemPlaceholder '%v'", err))
	}

	err = tx.Commit()
	if err != nil {
		panic(fmt.Sprintf("Failed at CreateItemPlaceholder '%v'", err))
	}
	return
}

// CreateItemPlaceholder creates a unique record in files table, so it hosds a parent/name constraint
// Since not record created in changelog, other user wont be able to see this file until import is finished.
func (m *filesDB) CreateItemPlaceholder(parentID int64, name string) (id int64, err error) {
	fileExt := path.Ext(name)
	fileName := name[0 : len(name)-len(fileExt)]
	id = 0
	for suffix := 0; suffix < 100; suffix++ {
		if suffix > 0 {
			name = fmt.Sprintf("%s-%d.%s", fileName, suffix, fileExt)
		}
		id, err = m.dbCreateItemPlaceholder(parentID, name)
		if err == nil {
			return
		}
	}
	return
}

func (m *filesDB) dbDeleteItemPlaceholder(id int64) {
	tx, err := m.database.Begin()
	if err != nil {
		panic(fmt.Sprintf("Failed at DeleteItemPlaceholder '%v'", err))
	}

	tx.Exec("delete from files where id = $1", id)

	err = tx.Commit()
	if err != nil {
		panic(fmt.Sprintf("Failed at DeleteItemPlaceholder '%v'", err))
	}
}

func (m *filesDB) DeleteItemPlaceholder(id int64) {
	m.dbDeleteItemPlaceholder(id)
}

func (m *filesDB) dbImportItem(itemID int64, fm *fileMeta) (err error) {
	tx, err := m.database.Begin()
	if err != nil {
		return
	}

	_, err = tx.Exec(
		"update files set scan_time = $1, size = $2, mdate = $3, cdate = $4, ctype = $5 where id = $6",
		m.startTime, fm.Size, fm.MDate, fm.CDate, fm.CType, itemID)
	if err != nil {
		tx.Rollback()
		return
	}

	var parentID int64
	row := tx.QueryRow(`SELECT parent_id FROM files where id = ?`, itemID)
	if err = row.Scan(&parentID); err != nil {
		tx.Rollback()
		return
	}

	_, err = tx.Exec(
		"insert into changelog (parent_id, file_id, action) values ($1, $2, $3)",
		parentID, itemID, actionAdd)
	if err != nil {
		tx.Rollback()
		return
	}

	tx.Commit()
	return
}

func (m *filesDB) ImportItem(itemID int64, itemPath string) (err error) {
	item, err := createFileItem(itemPath)
	if err != nil {
		return
	}
	err = m.dbImportItem(itemID, item.fileMeta())
	return
}

func (m *filesDB) RemoveItem(id int64) (err error) {
	err = m.dbRemoveFile(id)
	return
}

func (m *filesDB) dbMoveFile(id int64, newParent int64) {

}

func (m *filesDB) ScanPath(p string) int64 {
	p = path.Clean(p)
	log.Printf("Scanning %s ... ", p)

	if !strings.HasPrefix(p, "/") {
		log.Printf("Not a path '%v'", p)
		return -1
	}

	// find common parent
	pathComponents := strings.Split(p, "/")
	var cPath string
	parentID := m.rootID

	var i int
	for i = 1; i < len(pathComponents)-1; i++ {
		row := m.database.QueryRow(`SELECT id FROM files WHERE name = ? AND parent_id = ?`, pathComponents[i], parentID)
		if err := row.Scan(&parentID); err != nil {
			break
		}
	}
	cPath = path.Join(pathComponents[0:i]...)
	pathComponents = pathComponents[i:]

	tx, err := m.database.Begin()
	if err != nil {
		log.Print(err)
		return -1
	}

	stmt, _ := tx.Prepare(`
		insert into files (parent_id, scan_time, size, mdate, cdate, name, ctype)
   			values ($1, $2, $3, $4, $5, $6, $7)
   		on conflict (parent_id, name) do
   			update set scan_time=$2, size=$3, mdate=$4, cdate=$5, ctype=$7
   			where parent_id=$1 and name=$6;
		`)
	updateOrCreateItem := func(parentID int64, fm *fileMeta) (int64, error) {
		_, err := tx.Stmt(stmt).Exec(parentID, m.startTime, fm.Size, fm.MDate, fm.CDate, fm.Name, fm.CType)
		if err != nil {
			return -1, err
		}
		var id int64
		row := tx.QueryRow(`SELECT id FROM files WHERE parent_id=$1 AND name=$2`, parentID, fm.Name)
		if err := row.Scan(&id); err != nil {
			return -1, err
		}
		return id, nil
	}

	var f *fileItem

	// extend path
	cPath = path.Join("/", cPath)
	for _, c := range pathComponents { // we always run into at least once
		cPath = path.Join(cPath, c)
		if f, err = createFileItem(cPath); err != nil {
			log.Printf("Terminating at '%s' %v", cPath, err)
			tx.Rollback()
			return -1
		}
		if parentID, err = updateOrCreateItem(parentID, f.fileMeta()); err != nil {
			log.Printf("Terminating at '%s':x %v", cPath, err)
			tx.Rollback()
			return -1
		}
	}

	// refresh subtree
	var walkFileItem func(*fileItem, int64)
	walkFileItem = func(f *fileItem, parentID int64) {
		if f.fi.Mode().IsDir() {
			dir, err := os.Open(f.path)
			if err != nil {
				log.Print(err)
				return
			}
			list, err := dir.Readdir(-1)
			if err != nil {
				log.Print(err)
				return
			}

			for _, item := range list {
				cPath = path.Join(f.path, item.Name())
				child, err := createFileItem(cPath)
				if err != nil {
					log.Printf("Skipping '%s': %v", cPath, err)
				} else {
					childID, err := updateOrCreateItem(parentID, child.fileMeta())
					if err != nil {
						log.Printf("Error at '%s': %v", cPath, err)
					} else {
						walkFileItem(child, childID)
					}
				}

			}
		}
	}
	walkFileItem(f, parentID)

	// Clean orphaned items (cascade deletion)
	log.Printf("Cleaning orphans ... ")
	_, err = tx.Exec(`DELETE FROM files WHERE parent_id=$1 and scan_time<$2`, parentID, m.startTime)
	if err != nil {
		log.Print(err)
		tx.Rollback()
	}

	tx.Commit()
	log.Printf("Done")

	return parentID
}

func (m *filesDB) GetPathForID(id int64) (string, error) {
	p := ""
	for id != m.rootID {
		var c string
		row := m.database.QueryRow(`SELECT name, parent_id FROM files WHERE id = ?`, id)
		if err := row.Scan(&c, &id); err != nil {
			return "", err
		}
		p = path.Join(c, p)
	}
	return path.Join("/", p), nil
}

func (m *filesDB) GetIDForPath(p string) (int64, error) {
	p = path.Clean(p)
	if !path.IsAbs(p) {
		return -1, errors.New("f-YOU")
	}

	id := m.rootID
	pc := strings.Split(p, "/")

	for i := 1; i < len(pc); i++ {
		row := m.database.QueryRow(`SELECT id FROM files WHERE parent_id=$1 AND name=$2`, id, pc[i])
		if err := row.Scan(&id); err != nil {
			return -1, err
		}
	}

	return id, nil
}

func (m *filesDB) GetMetaDataForItemWithID(id int64) interface{} {
	fm := new(fileMeta)

	row := m.database.QueryRow(`
		SELECT 	id,
				CASE ctype 
					WHEN $1 THEN (SELECT count(*) FROM files AS f_size WHERE f_size.parent_id=files.id)
					ELSE size
				END item_size, 
				mdate, cdate, name, ctype 
		FROM files WHERE id=$2`,
		contentTypeDirectory, id)
	if err := row.Scan(&fm.ID, &fm.Size, &fm.MDate, &fm.CDate, &fm.Name, &fm.CType); err != nil {
		return nil
	}

	return fm
}

func (m *filesDB) GetChangesInDirectorySince(id int64, syncAnchor int64, count int) interface{} {
	changes, err := m.database.Query(`
		SELECT  changelog.id, changelog.file_id, changelog.action, 
				files.name, files.ctype, files.mdate, files.cdate,
				CASE ctype 
					WHEN $1 THEN (SELECT count(*) FROM files AS f_size WHERE f_size.parent_id=files.id)
					ELSE size
				END item_size
				FROM changelog LEFT JOIN files ON changelog.file_id = files.id 
				WHERE changelog.parent_id = $2 AND changelog.id > $3
				ORDER BY changelog.id ASC
				LIMIT $4`,
		contentTypeDirectory, id, syncAnchor, count)

	if err != nil {
		log.Printf("%v", err)
		return nil
	}

	result := struct {
		New    []*fileMeta `json:"new"`
		Erase  []int64     `json:"erase"`
		Anchor int64       `json:"anchor"`
		Remain int         `json:"remain"`
	}{
		New:    make([]*fileMeta, 0, count),
		Erase:  make([]int64, 0, count),
		Anchor: syncAnchor,
		Remain: 0,
	}

	for changes.Next() {
		fm := new(fileMeta)
		var action int64
		var name, ctype sql.NullString
		var mdate, cdate, size sql.NullInt64
		if err := changes.Scan(&result.Anchor, &fm.ID, &action, &name, &ctype, &mdate, &cdate, &size); err != nil {
			log.Printf("%v", err)
			return nil
		}

		switch action {
		case actionAdd:
			fm.Name = name.String
			fm.CType = ctype.String
			fm.MDate = mdate.Int64
			fm.CDate = cdate.Int64
			fm.Size = size.Int64
			result.New = append(result.New, fm)
			break
		case actionErase:
			result.Erase = append(result.Erase, fm.ID)
			break
		default:
			continue
		}
	}

	recordsLeft := m.database.QueryRow(`
		SELECT count(*) FROM changelog
		WHERE parent_id = $1 and id > $2`,
		id, result.Anchor)

	if err := recordsLeft.Scan(&result.Remain); err != nil {
		log.Printf("%v", err)
		return nil
	}

	return result
}
