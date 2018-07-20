package gormdb

// # Database config
// db.driver=sqlite3 # mysql, postgres, sqlite3
// db.host=localhost  # Use dbhost  /tmp/app.db is your driver is sqlite
// db.port=dbport
// db.user=dbuser
// db.name=dbname
// db.password=dbpassword
// db.singulartable=false # default=false

import (
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"    // mysql package
	_ "github.com/jinzhu/gorm/dialects/postgres" // postgres package
	_ "github.com/jinzhu/gorm/dialects/sqlite"   // mysql package
	"github.com/revel/revel"
)

// DB Gorm
var (
	DB      *gorm.DB
	gormLog = revel.AppLog
)

func init() {
	revel.RegisterModuleInit(func(m *revel.Module) {
		gormLog = m.Log
	})
}

// InitDB database
func OpenDB(dbDriver string, dbInfo string) {
	db, err := gorm.Open(dbDriver, dbInfo)
	if err != nil {
		gormLog.Fatal("sql.Open failed", "error", err)
	}
	DB = db
	singulartable := revel.Config.BoolDefault("db.singulartable", false)
	if singulartable {
		DB.SingularTable(singulartable)
	}

}

type DbInfo struct {
	DbDriver   string
	DbHost     string
	DbPort     int
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
		dbInfo = fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=disable password=%s", params.DbHost, params.DbPort, params.DbUser, params.DbName, params.DbPassword)
	case "mysql":
		dbInfo = fmt.Sprintf("%s:%s@(%s:%d)/%s?charset=utf8&parseTime=True&loc=Local", params.DbUser, params.DbPassword, params.DbHost, params.DbPort, params.DbName)
		fmt.Println(dbInfo)
	}
	OpenDB(params.DbDriver, dbInfo)

}

func InitDB() {
	params := DbInfo{}
	params.DbDriver = revel.Config.StringDefault("db.driver", "sqlite3")
	params.DbHost = revel.Config.StringDefault("db.host", "localhost")

	switch params.DbDriver {
	case "postgres":
		params.DbPort = revel.Config.IntDefault("db.port", 5432)
	case "mysql":
		params.DbPort = revel.Config.IntDefault("db.port", 3306)
	case "sqlite3":
		if params.DbHost == "localhost" {
			params.DbHost = "/tmp/app.db"
		}
	}

	params.DbUser = revel.Config.StringDefault("db.user", "default")
	params.DbPassword = revel.Config.StringDefault("db.password", "")
	params.DbName = revel.Config.StringDefault("db.name", "default")

	InitDBWithParameters(params)
}
