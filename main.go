package main

import (
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	_ "github.com/go-sql-driver/mysql"
	"os"
	"pal-importer/models/orm"
	"pal-importer/service"
	"time"
)

func init() {
	initOrm()
	startWorkers()
}

func initOrm() {
	orm.DefaultTimeLoc = time.UTC
	orm.RegisterDriver("mysql", orm.DRMySQL)
	var hostname string
	var err0 error

	hostname, err0 = os.Hostname()
	if (err0 == nil) && (hostname == "joachimmartillo-XPS-8700") {
		orm.RegisterDataBase("default", "mysql", "root:BRidge6-5094@/ORM_TEST?charset=utf8")
	}
	if (err0 == nil) && (hostname == "algotrader-XPS-13-9350") {
		orm.RegisterDataBase("default", "mysql", "root:BRidge6-5094@/ORM_TEST?charset=utf8")
	}
	registerModels()

	if _, err := orm.GetDB(); err != nil { // need to make work with dummy databases.
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
	orm.RegisterModel(
		new(modelsOrm.PalSpace),
		new(modelsOrm.FileInPal),
		new(modelsOrm.ClassificationInPal),
		new(modelsOrm.TagsInLibraries))
}

func startWorkers() {
	go new(service.WorkerMainInstance).Start(nil)
}

func main() {
	beego.Run()
}
