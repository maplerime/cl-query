/**
 * Licensed Materials - Property of PEG TECH INC
 *
 * (C) Copyright PEG TECH INC. 2019 ~ 2025 All Rights Reserved
 *
 * Contributors:
 *    bryan@raksmart.com - Initial implementation
 *
 *
 * Purpose: base database operation
 *
**/

package dbs

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"

	"github.com/maplerime/cl-query/pkg/conf"
	"github.com/maplerime/cl-query/utils/logging"
)

var (
	locker          = sync.Mutex{}
	dbm             *gorm.DB
	testMode        = strings.HasSuffix(os.Args[0], ".test") || os.Getenv("db.testing") != ""
	objects         = []interface{}{}
	migrationErrors = map[string]string{}
	upgradeErrors   = map[string]string{}
	grades          = map[string]func(*gorm.DB) error{}
	needToUpgrade   = false
	needToMigrate   = false
	OpenDB          = openDB
	logger          = logging.MustGetLogger("dbs")
	cfg             *conf.DBConfig
)

func InitDB(conf *conf.DBConfig) {
	logger.Debugf("Initializing database ... {%+v}", conf)
	cfg = conf
}

func openDB() (db *gorm.DB) {
	logger.Debugf("Opening database connection ...")
	dbType := cfg.GetType()
	dbUrl := cfg.GetUri()
	dbDebug := cfg.IsDebug()
	logger.Debugf("Database type: %s, URL: %s, Debug: %t", dbType, dbUrl, dbDebug)
	if testMode {
		dbType = "sqlite3"
		dbUrl = "file::memory:?cache=shared"
	}
	if dbType == "" {
		dbType = "sqlite3"
		dbUrl = "cland.db"
	}
	var err error
	if db, err = gorm.Open(dbType, dbUrl); err != nil {
		panic(err)
	}

	if testMode || dbDebug {
		db.LogMode(true)
	}

	// SetMaxIdleConns sets the maximum number of connections
	// in the idle connection pool.
	idle := cfg.GetIdle()
	db.DB().SetMaxIdleConns(idle)

	// SetMaxOpenConns sets the maximum number of open connections
	// to the database.
	open := cfg.GetOpen()
	db.DB().SetMaxOpenConns(open)

	// SetConnMaxLifetime set max connection lifetime(in minite)
	lifetime := cfg.GetLifetime()
	db.DB().SetConnMaxLifetime(time.Minute * time.Duration(lifetime))
	return db
}

func newDB() *gorm.DB {
	logger.Debugf("Creating new database connection ...")
	locker.Lock()
	defer locker.Unlock()
	if dbm == nil {
		dbm = OpenDB()
	}
	doAutoMigrate(dbm)
	doAutoUpgrade(dbm)
	return dbm
}

func AutoMigrate(values ...interface{}) {
	logger.Debugf("Marking model %+v for auto migration ...", values)
	locker.Lock()
	defer locker.Unlock()
	objects = append(objects, values...)
	needToMigrate = true
}

func doAutoMigrate(db *gorm.DB) {
	logger.Debugf("Tracing auto migrating database ...")
	if needToMigrate {
		logger.Debugf("Auto migrating database ...")
		names := tableNames(db)
		for i := 0; i < len(objects); i++ {
			obj := objects[i]
			name := names[i]
			err := db.AutoMigrate(obj).Error
			if err != nil {
				logger.Errorf("Failed to auto migrate: %v", err)
				msg := err.Error()
				if s, ok := migrationErrors[name]; ok {
					migrationErrors[name] = fmt.Sprintf("%s\n%s", s, msg)
				} else {
					migrationErrors[name] = msg
				}
			}
		}
		needToMigrate = false
	}
	logger.Debugf("Auto migration completed")
}

func AutoUpgrade(name string, grade func(*gorm.DB) error) {
	locker.Lock()
	defer locker.Unlock()
	grades[name] = grade
	needToUpgrade = true
}

func doAutoUpgrade(db *gorm.DB) (err error) {
	logger.Debugf("Tracing auto upgrading database ...")
	if !needToUpgrade || len(grades) == 0 {
		logger.Debugf("No need to auto upgrade database")
		return
	}
	names := []string{}
	for name, _ := range grades {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		grade := grades[name]
		if grade == nil { // skip nil
			continue
		}
		if err = grade(db); err != nil {
			logger.Errorf("Failed to auto upgrade: %v", err)
			upgradeErrors[name] = err.Error()
			continue
		}
	}
	logger.Debugf("Auto upgrade completed")
	needToUpgrade = false
	return
}

func DB() *gorm.DB {
	if dbm != nil {
		if needToMigrate || needToUpgrade {
			locker.Lock()
			if needToMigrate {
				doAutoMigrate(dbm)
			}
			if needToUpgrade {
				doAutoUpgrade(dbm)
			}
			locker.Unlock()
		}
		return dbm
	}
	return newDB()
}

func SetDB(db *gorm.DB) {
	locker.Lock()
	defer locker.Unlock()
	if db == dbm {
		return
	}
	needToMigrate = true
	needToUpgrade = true
	dbm = db
}

func TableNames() (names []string) {
	if dbm == nil {
		return
	}
	names = tableNames(dbm)
	return
}

func tableNames(db *gorm.DB) (names []string) {
	for i := 0; i < len(objects); i++ {
		obj := objects[i]
		names = append(names, db.NewScope(obj).TableName())
	}
	return
}
