modules/gorm
===============

[Gorm](http://jinzhu.me/gorm) module

## Activation
```ini
module.gorm = github.com/revel/modules/gorm
```

## Drivers

* sqlite3
* postgres
* mysql

## Configuration file

```ini
# Database config
app.dbautoinit=true # default=true
app.dbdriver=sqlite # mysql, postgres, sqlite3
app.dbhost=localhost  # Use dbhost  /tmp/app.db is your driver is sqlite
app.dbuser=dbuser
app.dbname=dbname
app.dbpassword=dbpassword

```

