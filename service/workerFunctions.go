package service

import "github.com/astaxie/beego/orm"

type WorkerFunctions struct {
	ormer orm.Ormer
}

func (o *WorkerFunctions) GetOrmer() orm.Ormer {
	// initialize the orm if its not already initialized
	if o.ormer == nil {
		o.ormer = orm.NewOrm()
	}
	return o.ormer
}

func (o *WorkerFunctions) SetOrmer(ormer orm.Ormer) {
	o.ormer = ormer
}
