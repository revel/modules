modules/gorm
===============

[Gorm](http://jinzhu.me/gorm) module

## Activation
```ini
module.gorm = github.com/revel/modules/orm/gorm
```

## Drivers

* sqlite3
* postgres
* mysql

## Configuration file

```ini
# Database config
db.autoinit=true # default=true
db.driver=sqlite3 # mysql, postgres, sqlite3
db.host=/tmp/app.db  # Use db.host /tmp/app.db is your driver is sqlite
#db.port=dbport # when the port is not used the default of each driver
#db.user=dbuser
#db.name=dbname
#db.password=dbpassword
#db.singulartable=false # default=false
```

#### Database Configuration Parameters Extended Information
* _autoinit_: The `Db` is initialized from the app.conf if `db.autoinit=true`.
* _singulartable_: By default all tables created based on a struct are pluralized.
                   For Example: a `type User struct {}` becomes table `users` in the database
                   , by setting `singulartable` to `true`, User's default table name will be `user`.
                   __Note__ table names set with `TableName` won't be affected by this setting.
                   You can also change the created table names by setting gorm.DefaultTableNameHandler on AppStartup
                   or func init() see [here](http://jinzhu.me/gorm/models.html#conventions)  for more details


## Example usage with transactions
```go
package controllers

import (
    "github.com/revel/revel"
    gormc "github.com/revel/modules/orm/gorm/app/controllers"
)

type App struct {
    gormc.TxnController
}

type Toy struct {
    Name string
}

func (c App) Index() revel.Result {
    c.Txn.LogMode(true)
    c.Txn.AutoMigrate(&Toy{})
    c.Txn.Save(&Toy{Name: "Fidget spinner"})

    return c.Render()
}
```

## Example usage without transactions
```go
package controllers

import (
    "github.com/revel/revel"
    gormc "github.com/revel/modules/orm/gorm/app/controllers"
)

type App struct {
    gormc.Controller
}

type Toy struct {
    Name string
}

func (c App) Index() revel.Result {
    c.DB.LogMode(true)
    c.DB.AutoMigrate(&Toy{})
    c.DB.Save(&Toy{Name: "Fidget spinner"})

    return c.Render()
}
```
