package gorm

// # Database config
// app.dbdriver=sqlite3 # mysql, postgres, sqlite3
// app.dbhost=localhost  # Use dbhost  /tmp/app.dbdb is your driver is sqlite
// app.dbuser=dbuser
// app.dbname=dbname
// app.dbpassword=dbpassword

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"    // mysql package
	_ "github.com/jinzhu/gorm/dialects/postgres" // postgres package
	_ "github.com/jinzhu/gorm/dialects/sqlite"   // mysql package
	"github.com/revel/revel"
)

func checkErr(err error, msg string) {
	if err != nil {
		log.Fatalln(msg, err)
	}
}

// DB Gorm
var DB *gorm.DB

// InitDB database
func OpenDB(dbDriver string, dbInfo string) {
	db, err := gorm.Open(dbDriver, dbInfo)
	if err != nil {
		checkErr(err, "sql.Open failed")
	}
	DB = db
}

type DbInfo struct {
	DbDriver   string
	DbHost     string
	DbUser     string
	DbPassword string
	DbName     string
}

func InitDBWithParameters(params DbInfo) {
	dbInfo := ""
	switch params.DbDriver {
	default:
		dbInfo = fmt.Sprintf(params.DbHost)
	case "postgres":
		dbInfo = fmt.Sprintf("host=%s user=%s dbname=%s sslmode=disable password=%s", params.DbHost, params.DbUser, params.DbName, params.DbPassword)
	case "mysql":
		dbInfo = fmt.Sprintf("%s:%s@%s/%s?charset=utf8&parseTime=True&loc=Local", params.DbUser, params.DbPassword, params.DbHost, params.DbName)
	}
	OpenDB(params.DbDriver, dbInfo)

}

func InitDB() {
	params := DbInfo{}
	params.DbDriver = revel.Config.StringDefault("app.dbdriver", "sqlite3")
	params.DbHost = revel.Config.StringDefault("app.dbhost", "localhost")
	if params.DbDriver == "sqlite" && params.DbHost == "localhost" {
		params.DbHost = "/tmp/app.dbdb"
	}
	params.DbUser = revel.Config.StringDefault("app.dbuser", "default")
	params.DbPassword = revel.Config.StringDefault("app.dbpassword", "")
	params.DbName = revel.Config.StringDefault("app.dbname", "default")

	InitDBWithParameters(params)
}

// GormController controllers begin, commit and rollback transactions
type GormController struct {
	revel.Controller
	Txn *gorm.DB
}

func (c *GormController) InitDB(params DbInfo) {
	dbInfo := ""
	switch params.DbDriver {
	default:
		dbInfo = fmt.Sprintf(params.DbHost)
	case "postgres":
		dbInfo = fmt.Sprintf("host=%s user=%s dbname=%s sslmode=disable password=%s", params.DbHost, params.DbUser, params.DbName, params.DbPassword)
	case "mysql":
		dbInfo = fmt.Sprintf("%s:%s@%s/%s?charset=utf8&parseTime=True&loc=Local", params.DbUser, params.DbPassword, params.DbHost, params.DbName)
	}
	OpenDB(params.DbDriver, dbInfo)

}

// Begin GormController to connect db
func (c *GormController) Begin() revel.Result {

	txn := DB.Begin()
	if txn.Error != nil {
		fmt.Println(c.Txn.Error)
		panic(txn.Error)
	}

	c.Txn = txn
	return nil
}

// Commit database transaction
func (c *GormController) Commit() revel.Result {
	if c.Txn == nil {
		return nil
	}

	c.Txn.Commit()
	if c.Txn.Error != nil && c.Txn.Error != sql.ErrTxDone {
		fmt.Println(c.Txn.Error)
		panic(c.Txn.Error)
	}

	c.Txn = nil
	return nil
}

// Rollback transaction
func (c *GormController) Rollback() revel.Result {
	if c.Txn == nil {
		return nil
	}

	c.Txn.Rollback()
	if c.Txn.Error != nil && c.Txn.Error != sql.ErrTxDone {
		fmt.Println(c.Txn.Error)
		panic(c.Txn.Error)
	}

	c.Txn = nil
	return nil
}

func init() {
	revel.OnAppStart(func() {
		if revel.Config.BoolDefault("app.dbautoinit", false) {
			InitDB()
			revel.InterceptMethod((*GormController).Begin, revel.BEFORE)
			revel.InterceptMethod((*GormController).Commit, revel.AFTER)
			revel.InterceptMethod((*GormController).Rollback, revel.FINALLY)
		}
	})
}
