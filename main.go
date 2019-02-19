package main

import (
	_ "encoding/csv"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	_ "github.com/go-sql-driver/mysql"
	"os"
	"pal-importer/models/orm"
	"pal-importer/service"
	"time"
)

var excludeList []string

// go initializes each file with func init() -- init() is not enter
// into symbol table

func init() {
	initOrm()      // database initialization
	startWorkers() // the workers are needed for the
	// the asynchronous database and client interface
}

// This stuff all initializes the Beego Orm database subsystem.
func initOrm() {
	orm.DefaultTimeLoc = time.UTC
	orm.RegisterDriver("mysql", orm.DRMySQL)
	registerModels()

	if _, err := orm.GetDB(); err != nil {
		// Connect default DB
		driverName := os.Getenv("DB_DRIVER")
		dataSource := os.Getenv("DB_SOURCE")
		if driverName == "" {
			driverName = beego.AppConfig.String("dbDriver")
		}
		if dataSource == "" {
			dataSource = beego.AppConfig.String("dbSource")
		}
		maxIdle, _ := beego.AppConfig.Int("dbMaxIdle")
		maxConn, _ := beego.AppConfig.Int("dbMacConn")
		orm.RegisterDataBase("default", driverName, dataSource, maxIdle, maxConn)

		orm.DefaultTimeLoc = time.UTC
	}

	orm.Debug, _ = beego.AppConfig.Bool("ormDebug")
}

func registerModels() {
	orm.RegisterModel( // These are databases pal-importer uses
		new(modelsOrm.PalSpace),            // start with pal space
		new(modelsOrm.FileInPal),           // find file in pal
		new(modelsOrm.ClassificationInPal), // organize classification in pal -- pal protocol
		new(modelsOrm.TagsInLibraries))     // transform into TagsInLibraries
}

func startWorkers() {
	go new(service.WorkerMainInstance).Start(nil) // Here is where program really starts.
}

func main() {
	beego.Run() // get beego running -- This is boiler plate.
}
