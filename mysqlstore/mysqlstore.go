package mysqlstore

import (
	"fmt"
	"github.com/kierdavis/argo"
	"github.com/yanatan16/GoMySQL"
	"os"
	"strings"
)

type cachedLiteral struct {
	Value      string
	Language   string
	DatatypeID uint64
}

type mysqlRequest struct {
	Query      string
	Params     []interface{}
	ResultChan chan []interface{}
	FailChan   chan bool
}

func term2node(term argo.Term) (node string) {
	switch realterm := term.(type) {
	case *argo.Resource:
		return realterm.URI
	case *argo.BlankNode:
		return "_:" + realterm.ID
	}

	return ""
}

func node2term(node string) (term argo.Term) {
	if len(node) >= 2 && node[0] == '_' && node[1] == ':' {
		return argo.NewBlankNode(node[2:])
	} else {
		return argo.NewResource(node)
	}

	return nil
}

type MySQLStore struct {
	Debug        bool
	ErrorHandler func(error)

	requests         chan *mysqlRequest
	tablePrefix      string
	nodeLookup       map[string]uint64
	prefixLookup     map[string]uint64
	literalLookup    map[string]uint64
	nodeRevLookup    map[uint64]string
	prefixRevLookup  map[uint64]string
	literalRevLookup map[uint64]*cachedLiteral
}

func DefaultErrorHandler(err error) {
	fmt.Fprintf(os.Stderr, "MySQL Error: %s\n", err.Error())
	os.Exit(1)
}

func NewMySQLStore(host, user, passwd, database, tablePrefix string) (store *MySQLStore) {
	client, err := mysql.DialTCP(host, user, passwd, database)
	if err != nil {
		panic(err)
	}

	//client.LogLevel = 1

	store = &MySQLStore{
		Debug:        false,
		ErrorHandler: DefaultErrorHandler,

		requests:         make(chan *mysqlRequest),
		tablePrefix:      tablePrefix,
		nodeLookup:       make(map[string]uint64),
		prefixLookup:     make(map[string]uint64),
		literalLookup:    make(map[string]uint64),
		nodeRevLookup:    make(map[uint64]string),
		prefixRevLookup:  make(map[uint64]string),
		literalRevLookup: make(map[uint64]*cachedLiteral),
	}

	go store.handleMySQLRequests(client)

	return store
}

func (store *MySQLStore) handleMySQLRequest(client *mysql.Client, request *mysqlRequest) (err error) {
	query := fmt.Sprintf(request.Query, store.tablePrefix)

	if store.Debug {
		fmt.Printf("Executing: %s\n", fmt.Sprintf(strings.Replace(query, "?", "%v", -1), request.Params...))
	}

	stmt, err := client.Prepare(query)
	if err != nil {
		return err
	}

	defer stmt.Close()

	err = stmt.BindParams(request.Params...)
	if err != nil {
		return err
	}

	err = stmt.Execute()
	if err != nil {
		return err
	}

	nFields := stmt.FieldCount()
	results := make([]interface{}, nFields)
	resultptrs := make([]interface{}, nFields)

	for i := 0; i < int(nFields); i++ {
		resultptrs[i] = &(results[i])
	}

	if stmt.MoreResults() {
		err = stmt.BindResult(resultptrs...)
		if err != nil {
			return err
		}

		for {
			eof, err := stmt.Fetch()
			if err != nil {
				return err
			}
			if eof {
				break
			}

			request.ResultChan <- results
		}
	}

	return nil
}

func (store *MySQLStore) handleMySQLRequests(client *mysql.Client) {
	for {
		request := <-store.requests
		err := store.handleMySQLRequest(client, request)
		close(request.ResultChan)

		request.FailChan <- err != nil
		if err != nil {
			store.ErrorHandler(err)
		}

		close(request.FailChan)
	}
}

func (store *MySQLStore) execute(query string, params ...interface{}) (resultChan chan []interface{}, failChan chan bool) {
	resultChan = make(chan []interface{})
	failChan = make(chan bool)
	store.requests <- &mysqlRequest{query, params, resultChan, failChan}

	return resultChan, failChan
}

func (store *MySQLStore) selectOne(query string, params ...interface{}) (row []interface{}, ok bool) {
	// Either returns:
	//   (row, true) - success, returns the first valid row
	//   (nil, true) - success but no results
	//   (nil, false) - error, caller should return immediately

	resultChan, failChan := store.execute(query, params...)
	row = <-resultChan

	// Clean out extra results
	for _ = range resultChan {

	}

	if row == nil { // Channel has been closed, so either an error occured or no results were returned
		return nil, !<-failChan
	}

	return row, true
}

func (store *MySQLStore) execVoid(query string, params ...interface{}) (ok bool) {
	resultChan, failChan := store.execute(query, params...)

	// Clean out results
	for _ = range resultChan {

	}

	return !<-failChan
}

func (store *MySQLStore) cacheLookup(uri string, cache map[string]uint64, table string) (id uint64) {
	// Try the cache
	id, ok := cache[uri]
	if !ok { // Cache miss, try the database
		row, ok := store.selectOne("SELECT id FROM %s_"+table+" WHERE uri = ? LIMIT 1", uri)
		if !ok {
			return 0
		}

		if row == nil {
			if !store.execVoid("INSERT INTO %s_"+table+" SET uri = ?", uri) {
				return 0
			}

			row, ok = store.selectOne("SELECT id FROM %s_"+table+" WHERE uri = ? LIMIT 1", uri)
			if !ok {
				return 0
			}
		}

		id = row[0].(uint64)

		// Add it into the cache
		cache[uri] = id
	}

	fmt.Printf("cache lookup %q -> %d\n", uri, id)
	return id
}

func (store *MySQLStore) node2id(uri string) (id uint64) {
	return store.cacheLookup(uri, store.nodeLookup, "nodes")
}

func (store *MySQLStore) prefix2id(uri string) (id uint64) {
	return store.cacheLookup(uri, store.prefixLookup, "prefixes")
}

func (store *MySQLStore) literal2id(lit *argo.Literal) (id uint64) {
	datatypeURI := ""
	if lit.Datatype != nil {
		datatypeURI = lit.Datatype.(*argo.Resource).URI
	}

	hash := fmt.Sprintf("%s@%s^^%s", lit.Value, lit.Language, datatypeURI)

	// Try the cache
	id, ok := store.literalLookup[hash]
	if !ok { // Cache miss, try the database
		var datatypeID uint64
		if lit.Datatype != nil {
			datatypeID = store.node2id(datatypeURI)
		}

		row, ok := store.selectOne("SELECT id FROM %s_literals WHERE value = ? AND language = ? AND datatype = ? LIMIT 1", lit.Value, lit.Language, datatypeID)
		if !ok {
			return 0
		}

		if row == nil {
			if !store.execVoid("INSERT INTO %s_literals SET value = ? AND language = ? AND datatype = ?", lit.Value, lit.Value, datatypeID) {
				return 0
			}

			row, ok = store.selectOne("SELECT id FROM %s_literals WHERE value = ? AND language = ? AND datatype = ? LIMIT 1", lit.Value, lit.Language, datatypeID)
			if !ok {
				return 0
			}
		}

		id = row[0].(uint64)

		// Add it into the cache
		store.literalLookup[hash] = id
	}

	return id
}

func (store *MySQLStore) cacheRevLookup(id uint64, cache map[uint64]string, table string) (uri string) {
	uri, ok := cache[id]
	if !ok {
		row, ok := store.selectOne("SELECT uri FROM %s_"+table+" WHERE id = ?", id)
		if !ok || row == nil {
			return ""
		}

		uri = row[0].(string)
		cache[id] = uri
	}

	return uri
}

func (store *MySQLStore) id2node(id uint64) (uri string) {
	return store.cacheRevLookup(id, store.nodeRevLookup, "nodes")
}

func (store *MySQLStore) id2prefix(id uint64) (uri string) {
	return store.cacheRevLookup(id, store.prefixRevLookup, "prefixes")
}

func (store *MySQLStore) id2literal(id uint64) (term argo.Term) {
	cachedLit, ok := store.literalRevLookup[id]
	if !ok {
		row, ok := store.selectOne("SELECT value, language, datatype FROM %s_literals WHERE id = ?", id)
		if !ok || row == nil {
			return nil
		}

		cachedLit = &cachedLiteral{
			Value:      row[0].(string),
			Language:   row[1].(string),
			DatatypeID: row[2].(uint64),
		}

		store.literalRevLookup[id] = cachedLit
	}

	var datatype argo.Term = nil

	if cachedLit.DatatypeID != 0 {
		datatypeURI := store.id2node(cachedLit.DatatypeID)
		datatype = argo.NewResource(datatypeURI)
	}

	return argo.NewLiteralWithLanguageAndDatatype(cachedLit.Value, cachedLit.Language, datatype)
}

func (store *MySQLStore) CreateTables() {
	if !store.execVoid("CREATE TABLE IF NOT EXISTS %s_triples ( id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY, subject BIGINT UNSIGNED NOT NULL, predicatePrefix BIGINT UNSIGNED NOT NULL, predicateLocal VARCHAR(64) NOT NULL, objectIsLiteral TINYINT UNSIGNED NOT NULL, object BIGINT UNSIGNED NOT NULL )") {
		return
	}

	if !store.execVoid("CREATE TABLE IF NOT EXISTS %s_nodes ( id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY, uri VARCHAR(512) NOT NULL )") {
		return
	}

	if !store.execVoid("CREATE TABLE IF NOT EXISTS %s_literals ( id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY, value TEXT NOT NULL, language VARCHAR(32) NULL, datatype BIGINT UNSIGNED NULL )") {
		return
	}

	if !store.execVoid("CREATE TABLE IF NOT EXISTS %s_prefixes ( id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY, uri VARCHAR(512) NOT NULL )") {
		return
	}
}

func (store *MySQLStore) DropTables() {
	if !store.execVoid("DROP TABLE IF EXISTS %s_triples") {
		return
	}

	if !store.execVoid("DROP TABLE IF EXISTS %s_nodes") {
		return
	}

	if !store.execVoid("DROP TABLE IF EXISTS %s_literals") {
		return
	}

	if !store.execVoid("DROP TABLE IF EXISTS %s_prefixes") {
		return
	}
}

func (store *MySQLStore) encodeSubject(term argo.Term) (subjectID uint64) {
	return store.node2id(term2node(term))
}

func (store *MySQLStore) encodePredicate(term argo.Term) (predicatePrefixID uint64, predicateLocal string) {
	predicatePrefix, predicateLocal := argo.SplitPrefix(term.(*argo.Resource).URI)
	return store.prefix2id(predicatePrefix), predicateLocal
}

func (store *MySQLStore) encodeObject(term argo.Term) (objectIsLiteral uint8, objectID uint64) {
	lit, isLit := term.(*argo.Literal)
	if isLit {
		objectIsLiteral = 1
		objectID = store.literal2id(lit)

	} else {
		objectIsLiteral = 0
		objectID = store.node2id(term2node(term))
	}

	return objectIsLiteral, objectID
}

func (store *MySQLStore) encodeTriple(triple *argo.Triple) (subjectID uint64, predicatePrefixID uint64, predicateLocal string, objectIsLiteral uint8, objectID uint64) {
	subjectID = store.encodeSubject(triple.Subject)
	predicatePrefixID, predicateLocal = store.encodePredicate(triple.Predicate)
	objectIsLiteral, objectID = store.encodeObject(triple.Object)
	return subjectID, predicatePrefixID, predicateLocal, objectIsLiteral, objectID
}

func (store *MySQLStore) decodeSubject(subjectID uint64) (subject argo.Term) {
	return node2term(store.id2node(subjectID))
}

func (store *MySQLStore) decodePredicate(predicatePrefixID uint64, predicateLocal string) (predicate argo.Term) {
	predicatePrefix := store.id2prefix(predicatePrefixID)
	return argo.NewResource(predicatePrefix + predicateLocal)
}

func (store *MySQLStore) decodeObject(objectIsLiteral uint8, objectID uint64) (object argo.Term) {
	if objectIsLiteral != 0 {
		object = store.id2literal(objectID)
	} else {
		object = node2term(store.id2node(objectID))
	}

	return object
}

func (store *MySQLStore) decodeTriple(subjectID uint64, predicatePrefixID uint64, predicateLocal string, objectIsLiteral uint8, objectID uint64) (triple *argo.Triple) {
	return argo.NewTriple(store.decodeSubject(subjectID), store.decodePredicate(predicatePrefixID, predicateLocal), store.decodeObject(objectIsLiteral, objectID))
}

func (store *MySQLStore) Add(triple *argo.Triple) (index int) {
	subjectID, predicatePrefixID, predicateLocal, objectIsLiteral, objectID := store.encodeTriple(triple)

	if !store.execVoid("INSERT INTO %s_triples SET subject = ? AND predicatePrefix = ? AND predicateLocal = ? AND objectIsLiteral = ? AND object = ?", subjectID, predicatePrefixID, predicateLocal, objectIsLiteral, objectID) {
		return 0
	}

	row, ok := store.selectOne("SELECT id FROM %s_triples WHERE subject = ? AND predicatePrefix = ? AND predicateLocal = ? AND objectIsLiteral = ? AND object = ?", subjectID, predicatePrefixID, predicateLocal, objectIsLiteral, objectID)
	if !ok || row == nil {
		return 0
	}

	return int(row[0].(uint64))
}

func (store *MySQLStore) Remove(triple *argo.Triple) {
	subjectID, predicatePrefixID, predicateLocal, objectIsLiteral, objectID := store.encodeTriple(triple)

	if !store.execVoid("DELETE FROM %s_triples WHERE subject = ? AND predicatePrefix = ? AND predicateLocal = ? AND objectIsLiteral = ? AND object = ?", subjectID, predicatePrefixID, predicateLocal, objectIsLiteral, objectID) {
		return
	}
}

func (store *MySQLStore) RemoveIndex(index int) {
	if !store.execVoid("DELETE FROM %s_triples WHERE id = ?", index) {
		return
	}
}

func (store *MySQLStore) Clear() {
	store.DropTables()
	store.CreateTables()
}

func (store *MySQLStore) Num() (n int) {
	row, ok := store.selectOne("SELECT COUNT(*) FROM %s_triples")
	if !ok || row == nil {
		return 0
	}

	return int(row[0].(uint64))
}

func (store *MySQLStore) IterTriples() (ch chan *argo.Triple) {
	return store.Filter(nil, nil, nil)
}

func (store *MySQLStore) Filter(subjSearch, predSearch, objSearch argo.Term) (ch chan *argo.Triple) {
	ch = make(chan *argo.Triple)

	queryClauses := make([]string, 0)
	queryValues := make([]interface{}, 0)

	if subjSearch != nil {
		subjectID := store.encodeSubject(subjSearch)

		queryClauses = append(queryClauses, "subject = ?")
		queryValues = append(queryValues, subjectID)
	}

	if predSearch != nil {
		predicatePrefixID, predicateLocal := store.encodePredicate(predSearch)

		queryClauses = append(queryClauses, "predicatePrefix = ?")
		queryClauses = append(queryClauses, "predicateLocal = ?")
		queryValues = append(queryValues, predicatePrefixID)
		queryValues = append(queryValues, predicateLocal)
	}

	if objSearch != nil {
		objectIsLiteral, objectID := store.encodePredicate(objSearch)

		queryClauses = append(queryClauses, "objectIsLiteral = ?")
		queryClauses = append(queryClauses, "object = ?")
		queryValues = append(queryValues, objectIsLiteral)
		queryValues = append(queryValues, objectID)
	}

	whereStr := ""
	if len(queryClauses) > 0 {
		whereStr = " WHERE " + strings.Join(queryClauses, " AND ")
	}

	resultChan, failChan := store.execute("SELECT subject, predicatePrefix, predicateLocal, objectIsLiteral, object FROM %s_triples"+whereStr, queryValues...)

	go func() {
		for row := range resultChan {
			var subject, predicate, object argo.Term

			if subjSearch == nil {
				subject = store.decodeSubject(row[0].(uint64))
			} else {
				subject = subjSearch
			}

			if predSearch == nil {
				predicate = store.decodePredicate(row[1].(uint64), row[2].(string))
			} else {
				predicate = predSearch
			}

			if objSearch == nil {
				object = store.decodeObject(row[3].(uint8), row[4].(uint64))
			} else {
				object = objSearch
			}

			ch <- argo.NewTriple(subject, predicate, object)
		}

		<-failChan
		close(ch)
	}()

	return ch
}
