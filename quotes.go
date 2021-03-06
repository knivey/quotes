package quotes

import (
	"database/sql"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	sqlCreateTable = `CREATE TABLE IF NOT EXISTS quotes (` +
		`id INTEGER PRIMARY KEY,` +
		`date INTEGER NOT NULL,` +
		`author TEXT NOT NULL,` +
		`quote TEXT NOT NULL);`
	sqlDateIndex = `CREATE INDEX IF NOT EXISTS quotesdate ON quotes (date);`
	sqlGetCount  = `SELECT COUNT(*) FROM quotes;`
	sqlAdd       = `INSERT INTO quotes (date, author, quote) VALUES(?, ?, ?);`
	sqlDel       = `DELETE FROM quotes WHERE id = ?;`
	sqlEdit      = `UPDATE quotes SET quote = ? WHERE id = ?;`
	sqlGet       = `SELECT id, quote FROM quotes ORDER BY RANDOM() LIMIT 1;`
	sqlGetId     = `SELECT quote FROM quotes WHERE id = ?;`
	sqlGetDetail = `SELECT date, author FROM quotes WHERE id = ?;`
	sqlGetAll    = `SELECT id, date, author, quote FROM quotes order by id desc;`
)

// QuoteDB provides file storage of quotes via an sqlite database.
type QuoteDB struct {
	db      *sql.DB
	nQuotes int
	sync.RWMutex
}

// Quote is for serializing to and from the sqlite database.
type Quote struct {
	ID     int
	Date   time.Time
	Author string
	Quote  string
}

// OpenDB opens the database at the location requested.
func OpenDB(filename string) (*QuoteDB, error) {
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		return nil, err
	}

	qdb := &QuoteDB{db: db}
	err = qdb.createTable()
	if err != nil {
		defer qdb.Close()
		return nil, err
	}
	err = qdb.getCount()
	if err != nil {
		defer qdb.Close()
		return nil, err
	}

	return qdb, nil
}

// NQuotes returns the number of quotes in the database.
func (q *QuoteDB) NQuotes() int {
	q.RLock()
	defer q.RUnlock()
	return q.nQuotes
}

// createTableIfNotExists creates the quotes table if necessary.
func (q *QuoteDB) createTable() (err error) {
	_, err = q.db.Exec(sqlCreateTable)
	if err != nil {
		return
	}
	_, err = q.db.Exec(sqlDateIndex)
	return
}

// getCount refreshes the number of quotes.
func (q *QuoteDB) getCount() error {
	return q.db.QueryRow(sqlGetCount).Scan(&q.nQuotes)
}

// Close the database file.
func (q *QuoteDB) Close() error {
	err := q.db.Close()
	q.db = nil
	return err
}

// AddQuote adds a quote to the database.
func (q *QuoteDB) AddQuote(author, quote string) (id int64, err error) {
	q.Lock()
	defer q.Unlock()

	var res sql.Result
	res, err = q.db.Exec(sqlAdd, time.Now().Unix(), author, quote)
	if err != nil {
		return
	}

	if id, err = res.LastInsertId(); err != nil {
		id = 0
	}

	q.nQuotes++
	return
}

// RandomQuote gets a random existing quote.
func (q *QuoteDB) RandomQuote() (id int, quote string, err error) {
	err = q.db.QueryRow(sqlGet).Scan(&id, &quote)
	return
}

// GetQuote gets a specific quote by id.
func (q *QuoteDB) GetQuote(id int) (quote string, err error) {
	err = q.db.QueryRow(sqlGetId, id).Scan(&quote)
	return
}

// GetDetails gets metadata about the quote.
func (q *QuoteDB) GetDetails(id int) (date int64, author string, err error) {
	err = q.db.QueryRow(sqlGetDetail, id).Scan(&date, &author)
	return
}

// DelQuote deletes a quote by id.
func (q *QuoteDB) DelQuote(id int) (bool, error) {
	var err error
	var res sql.Result
	var r int64
	if res, err = q.db.Exec(sqlDel, id); err != nil {
		return false, err
	}
	if r, err = res.RowsAffected(); err != nil {
		return false, err
	}
	if r == 1 {
		q.Lock()
		defer q.Unlock()
		q.nQuotes--
		return true, nil
	}
	return false, nil
}

// EditQuote edits a quote by id.
func (q *QuoteDB) EditQuote(id int, quote string) (bool, error) {
	var err error
	var res sql.Result
	var r int64
	if res, err = q.db.Exec(sqlEdit, quote, id); err != nil {
		return false, err
	}
	if r, err = res.RowsAffected(); err != nil {
		return false, err
	}
	return r == 1, nil
}

func (qdb *QuoteDB) GetAll() ([]Quote, error) {
	var err error

	rows, err := qdb.db.Query(sqlGetAll)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	quotes := make([]Quote, 0)
	q := Quote{}
	for rows.Next() {
		var date int64
		if err = rows.Scan(&q.ID, &date, &q.Author, &q.Quote); err != nil {
			return nil, err
		}

		q.Date = time.Unix(date, 0).UTC()

		quotes = append(quotes, q)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return quotes, nil
}
