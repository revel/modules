package gorp

import (
	"database/sql"
	"github.com/revel/revel"
	"github.com/revel/revel/logger"
	sq "github.com/Masterminds/squirrel"
	"github.com/go-gorp/gorp"
)

// DB Gorp
type DbGorp struct {
	Map *gorp.DbMap
	// The Sql statement builder to use to build select statements
	SqlStatementBuilder sq.StatementBuilderType
	// Database connection information
	Info *DbInfo
	// The database initialization function
	dbInitFn func(dbMap *DbGorp) error
}

type DbInfo struct {
	DbDriver     string
	DbHost       string
	DbUser       string
	DbPassword   string
	DbName       string
	DbSchema     string
	DbConnection string
	Dialect      gorp.Dialect
}
type (
	DbGeneric interface {
		Select(i interface{}, query string, args ...interface{}) ([]interface{}, error)
		Exec(query string, args ...interface{}) (sql.Result, error)
		Insert(list ...interface{}) error
		Update(list ...interface{}) (int64, error)
		Delete(i ...interface{}) (int64, error)
		SelectOne(holder interface{}, query string, args ...interface{}) error
	}
	DbReadable interface {
		Builder() sq.StatementBuilderType
		Get(i interface{}, keys ...interface{}) (interface{}, error)
		Select(i interface{}, builder sq.SelectBuilder) (l []interface{}, err error)
		SelectOne(i interface{}, builder sq.SelectBuilder) (err error)
		SelectInt(builder sq.SelectBuilder) (i int64, err error)
		GetMap() DbGeneric
		Schema() string
	}
	DbWriteable interface {
		DbReadable
		Insert(list ...interface{}) error
		Update(list ...interface{}) (int64, error)
		Delete(i ...interface{}) (int64, error)
		ExecUpdate(builder sq.UpdateBuilder) (r sql.Result, err error)
		ExecInsert(builder sq.InsertBuilder) (r sql.Result, err error)
	}
)

// OpenDb database
func (dbGorp *DbGorp) OpenDb() (err error) {
	db, err := sql.Open(dbGorp.Info.DbDriver, dbGorp.Info.DbConnection)
	if err != nil {
		moduleLogger.Fatal("Open Database Error", "error", err)
	}

	// Create the database map
	dbGorp.Map = &gorp.DbMap{Db: db, Dialect: dbGorp.Info.Dialect}

	revel.OnAppStop(func() {
		revel.RevelLog.Info("Closing the database (from module)")
		if err := dbGorp.Close(); err != nil {
			revel.AppLog.Error("Failed to close the database", "error", err)
		}
	})

	return dbGorp.dbInit()
}

// Create a new database connection and open it from this one
func (dbGorp *DbGorp) CloneDb(open bool) (newDb *DbGorp, err error) {
	dbInfo := *dbGorp.Info
	newDb = &DbGorp{Info: &dbInfo}
	newDb.dbInitFn = dbGorp.dbInitFn
	err = newDb.InitDb(open)

	return
}

// Close the database connection
func (dbGorp *DbGorp) Begin() (txn *Transaction, err error) {
	tx, err := dbGorp.Map.Begin()
	if err != nil {
		return
	}
	txn = &Transaction{tx, dbGorp}
	return
}

// Close the database connection
func (dbGorp *DbGorp) Close() (err error) {
	if dbGorp.Map.Db != nil {
		err = dbGorp.Map.Db.Close()
	}
	return
}

// Called to perform table registration and anything else that needs to be done on a new connection
func (dbGorp *DbGorp) dbInit() (err error) {
	if dbGorp.dbInitFn != nil {
		err = dbGorp.dbInitFn(dbGorp)
	}
	return
}

// Used to specifiy the init function to call when database is initialized
// Calls the init function immediately
func (dbGorp *DbGorp) SetDbInit(dbInitFn func(dbMap *DbGorp) error) (err error) {
	dbGorp.dbInitFn = dbInitFn
	return dbGorp.dbInit()
}

func (dbGorp *DbGorp) Select(i interface{}, builder sq.SelectBuilder) (l []interface{}, err error) {
	query, args, err := builder.ToSql()
	if err == nil {
		list, err := dbGorp.Map.Select(i, query, args...)
		if err != nil && gorp.NonFatalError(err) {
			return list, nil
		}
		if err == sql.ErrNoRows {
			err = nil
		}
		return list, err
	}
	return
}

func (dbGorp *DbGorp) SelectOne(i interface{}, builder sq.SelectBuilder) (err error) {
	query, args, err := builder.ToSql()
	if err == nil {
		err = dbGorp.Map.SelectOne(i, query, args...)
		if err != nil && gorp.NonFatalError(err) {
			return nil
		}
	}
	return
}

func (dbGorp *DbGorp) SelectInt(builder sq.SelectBuilder) (i int64, err error) {
	query, args, err := builder.ToSql()
	if err == nil {
		i, err = dbGorp.Map.SelectInt(query, args...)
	}
	return
}
func (dbGorp *DbGorp) ExecUpdate(builder sq.UpdateBuilder) (r sql.Result, err error) {
	query, args, err := builder.ToSql()
	if err == nil {
		r, err = dbGorp.Map.Exec(query, args...)
	}
	return
}
func (dbGorp *DbGorp) ExecInsert(builder sq.InsertBuilder) (r sql.Result, err error) {
	query, args, err := builder.ToSql()
	if err == nil {
		r, err = dbGorp.Map.Exec(query, args...)
	}
	return
}

//
// Shifted some common functions up a level
////

func (dbGorp *DbGorp) Insert(list ...interface{}) error {
	return dbGorp.Map.Insert(list...)
}

func (dbGorp *DbGorp) Update(list ...interface{}) (int64, error) {
	return dbGorp.Map.Update(list...)
}
func (dbGorp *DbGorp) Get(i interface{}, keys ...interface{}) (interface{}, error) {
	return dbGorp.Map.Get(i, keys...)
}
func (dbGorp *DbGorp) Delete(i ...interface{}) (int64, error) {
	return dbGorp.Map.Delete(i...)
}

func (dbGorp *DbGorp) TraceOn(log logger.MultiLogger) {
	dbGorp.Map.TraceOn("", &simpleTrace{log.New("section", "gorp")})

}
func (dbGorp *DbGorp) TraceOff() {

}
func (dbGorp *DbGorp) Builder() sq.StatementBuilderType {
	return dbGorp.SqlStatementBuilder
}
func (dbGorp *DbGorp) GetMap() DbGeneric {
	return dbGorp.Map
}
func (dbGorp *DbGorp) Schema() (result string) {
	if dbGorp.Info != nil {
		result = dbGorp.Info.DbSchema
	}
	return
}

type simpleTrace struct {
	log logger.MultiLogger
}

func (s *simpleTrace) Printf(format string, v ...interface{}) {
	s.log.Infof(format, v...)
}
