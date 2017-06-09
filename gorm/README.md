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
db.autoinit=true # default=true
db.driver=sqlite # mysql, postgres, sqlite3
db.host=localhost  # Use db.host /tmp/app.db is your driver is sqlite
db.user=dbuser
db.name=dbname
db.password=dbpassword

```

