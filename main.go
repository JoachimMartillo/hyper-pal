package main

import (
	"hyper-pal/service"
	"github.com/astaxie/beego/orm"
	"time"
	"os"
	"github.com/astaxie/beego"
	"hyper-pal/models/orm"
	_ "github.com/go-sql-driver/mysql"
)

func init() {
	initOrm()
	startWorkers()
}

func initOrm() {
	orm.DefaultTimeLoc = time.UTC
	orm.RegisterDriver("mysql", orm.DRMySQL)
	registerModels()

	if _, err := orm.GetDB(); err != nil {
		// Connect default DB
		driverName := os.Getenv("DB_DRIVER")
		dataSource := os.Getenv("DB_SOURCE")
		if (driverName == "") {
			driverName = beego.AppConfig.String("dbDriver")
		}
		if (dataSource == "") {
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
		new(modelsOrm.FileInPal))
}

func startWorkers() {
	go new (service.WorkerMainInstance).Start(nil)
}

func main() {
	beego.Run()
}
