package mysqlstore

import (
	"fmt"
	"github.com/kierdavis/argo"
	"github.com/yanatan16/GoMySQL"
)

const (
	TERM_INVALID = iota
	TERM_RESOURCE
	TERM_NODE
	TERM_LITERAL
)

type encodedTriple struct {
	st uint8
	sv string

	pv string

	ot uint8
	ov string
	ol string
	od string
}

type MySQLStore struct {
	Debug bool

	client *mysql.Client
	table  string
}

func NewMySQLStore(host, user, passwd, database, table string) (store *MySQLStore) {
	client, err := mysql.DialTCP(host, user, passwd, database)
	if err != nil {
		panic(err)
	}

	return &MySQLStore{Debug: false, client: client, table: table}
}

func (store *MySQLStore) execute(query string, params ...interface{}) (stmt *mysql.Statement, err error) {
	query = fmt.Sprintf(query, store.table)

	if store.Debug {
		fmt.Printf("Executing: %s\n", query)
	}

	stmt, err = store.client.Prepare(query)
	if err != nil {
		return nil, err
	}

	err = stmt.BindParams(params...)
	if err != nil {
		return nil, err
	}

	err = stmt.Execute()
	if err != nil {
		return nil, err
	}

	return stmt, nil
}

func (store *MySQLStore) CreateTable() {
	_, err := store.execute("CREATE TABLE IF NOT EXISTS %s ( id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY, subjtype TINYINT UNSIGNED NOT NULL, subjvalue VARCHAR(256), predvalue VARCHAR(256), objtype TINYINT UNSIGNED NOT NULL, objvalue VARCHAR(256), objlang VARCHAR(64), objdt VARCHAR(256) )")
	if err != nil {
		panic(err)
	}
}

func (store *MySQLStore) ReCreateTable() {
	_, err := store.execute("DROP TABLE IF EXISTS %s")
	if err != nil {
		panic(err)
	}

	store.CreateTable()
}

func (store *MySQLStore) encodeTriple(triple *argo.Triple) (enctriple *encodedTriple) {
	enctriple = new(encodedTriple)

	switch subject := triple.Subject.(type) {
	case *argo.Resource:
		enctriple.st = TERM_RESOURCE
		enctriple.sv = subject.URI

	case *argo.Node:
		enctriple.st = TERM_NODE
		enctriple.sv = subject.ID
	}

	enctriple.pv = triple.Predicate.(*argo.Resource).URI

	switch object := triple.Object.(type) {
	case *argo.Resource:
		enctriple.ot = TERM_RESOURCE
		enctriple.ov = object.URI

	case *argo.Literal:
		enctriple.ot = TERM_LITERAL
		enctriple.ov = object.Value

		if object.Language != "" {
			enctriple.ol = object.Language

		} else if object.Datatype != nil {
			enctriple.od = object.Datatype.(*argo.Resource).URI
		}

	case *argo.Node:
		enctriple.ot = TERM_NODE
		enctriple.ov = object.ID
	}

	return enctriple
}

func (store *MySQLStore) decodeTriple(enctriple *encodedTriple) (triple *argo.Triple) {
	var subj, pred, obj argo.Term

	switch enctriple.st {
	case TERM_RESOURCE:
		subj = argo.NewResource(enctriple.sv)

	case TERM_NODE:
		subj = argo.NewNode(enctriple.sv)

	default:
		panic(fmt.Errorf("Invalid subject type: %d", enctriple.st))
	}

	pred = argo.NewResource(enctriple.pv)

	switch enctriple.ot {
	case TERM_RESOURCE:
		obj = argo.NewResource(enctriple.ov)

	case TERM_NODE:
		obj = argo.NewNode(enctriple.ov)

	case TERM_LITERAL:
		if enctriple.ol != "" {
			obj = argo.NewLiteralWithLanguage(enctriple.ov, enctriple.ol)
		} else if enctriple.od != "" {
			obj = argo.NewLiteralWithDatatype(enctriple.ov, argo.NewResource(enctriple.od))
		} else {
			obj = argo.NewLiteral(enctriple.ov)
		}

	default:
		panic(fmt.Errorf("Invalid object type: %d", enctriple.ot))
	}

	return argo.NewTriple(subj, pred, obj)
}

func (store *MySQLStore) Add(triple *argo.Triple) (index int) {
	enctriple := store.encodeTriple(triple)
	stmt, err := store.execute("INSERT INTO %s VALUES ( NULL, ?, ?, ?, ?, ?, ?, ? )", enctriple.st, enctriple.sv, enctriple.pv, enctriple.ot, enctriple.ov, enctriple.ol, enctriple.od)
	if err != nil {
		panic(err)
	}
	stmt.Close()

	return int(stmt.LastInsertId)
}

func (store *MySQLStore) Remove(triple *argo.Triple) {
	enctriple := store.encodeTriple(triple)
	stmt, err := store.execute("DELETE FROM %s WHERE ( subjtype = ? AND subjvalue = ? AND predvalue = ? AND objtype = ? AND objvalue = ? AND objlang = ? AND objdt = ? )", enctriple.st, enctriple.sv, enctriple.pv, enctriple.ot, enctriple.ov, enctriple.ol, enctriple.od)
	if err != nil {
		panic(err)
	}
	stmt.Close()
}

func (store *MySQLStore) RemoveIndex(index int) {
	stmt, err := store.execute("DELETE FROM %s WHERE id = ?", index)
	if err != nil {
		panic(err)
	}
	stmt.Close()
}

func (store *MySQLStore) Clear() {
	store.ReCreateTable()
}

func (store *MySQLStore) Num() (n int) {
	stmt, err := store.execute("SELECT COUNT(*) FROM %s")
	if err != nil {
		panic(err)
	}

	err = stmt.BindResult(&n)
	if err != nil {
		panic(err)
	}

	_, err = stmt.Fetch()
	if err != nil {
		panic(err)
	}

	return n
}

func (store *MySQLStore) IterTriples() (ch chan *argo.Triple) {
	ch = make(chan *argo.Triple)

	go func() {
		stmt, err := store.execute("SELECT * FROM %s")
		if err != nil {
			panic(err)
		}

		var row *encodedTriple
		var id uint64

		err = stmt.BindResult(&id, &row.st, &row.sv, &row.pv, &row.ot, &row.ov, &row.ol, &row.od)
		if err != nil {
			panic(err)
		}

		for {
			eof, err := stmt.Fetch()
			if err != nil {
				panic(err)
			}

			if eof {
				break
			}

			ch <- store.decodeTriple(row)
		}
	}()

	return ch
}
