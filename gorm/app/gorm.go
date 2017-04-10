package gorm

// # Database config
// app.dbdriver=sqlite3 # mysql, postgres, sqlite3
// app.dbhost=localhost  # Use dbhost  /tmp/gorm.db is your driver is sqlite
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
func InitDB() {
	var dbDriver = revel.Config.StringDefault("app.dbdriver", "sqlite3")
	var dbHost = revel.Config.StringDefault("app.dbhost", "localhost")
	if dbDriver == "sqlite" && dbHost == "localhost" {
		dbHost = "/tmp/gorm.db"
	}
	var dbUser = revel.Config.StringDefault("app.dbuser", "postgres")
	var dbPassword = revel.Config.StringDefault("app.dbpassword", "")
	var dbName = revel.Config.StringDefault("app.dbname", "shopping")
	dbInfo := ""

	switch dbDriver {
	default:
		dbInfo = fmt.Sprintf(dbHost)
	case "postgres":
		dbInfo = fmt.Sprintf("host=%s user=%s dbname=%s sslmode=disable password=%s", dbHost, dbUser, dbName, dbPassword)
	case "mysql":
		dbInfo = fmt.Sprintf("%s:%s@%s/%s?charset=utf8&parseTime=True&loc=Local", dbUser, dbPassword, dbHost, dbName)
	}

	db, err := gorm.Open(dbDriver, dbInfo)
	if err != nil {
		checkErr(err, "sql.Open failed")
	}
	DB = db
}

// GormController controllers begin, commit and rollback transactions
type GormController struct {
	revel.Controller
	Txn *gorm.DB
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
	revel.OnAppStart(InitDB)
	revel.InterceptMethod((*GormController).Begin, revel.BEFORE)
	revel.InterceptMethod((*GormController).Commit, revel.AFTER)
	revel.InterceptMethod((*GormController).Rollback, revel.FINALLY)
}
