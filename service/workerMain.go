package service

import (
	"github.com/astaxie/beego/orm"
	"hyper-pal/models/orm"
	"time"
	"log"
	"fmt"
	"sync"
	"hyper-pal/models/hyper"
)

type WorkerMainInstance struct {
	WorkerFunctions

	onceGetHyper	sync.Once
	hyper			*Hyper
	onceGetPal		sync.Once
	pal				*AssetLibraryPhillips
}

func (o *WorkerMainInstance) Start(ormer orm.Ormer) {
	o.SetOrmer(ormer)
	//didSomething := false
	pauseTime := 5 * time.Second // 5 Seconds.
	updateRepeatTime := 1 * time.Hour // 1 Hour
	firstTime := true
	if err := modelsOrm.ClearPalSpaces(o.GetOrmer()); err != nil {
		log.Println(err.Error())
		return
	}

	// Forever
	for {
		// Delay a lit.
		if !firstTime {
			time.Sleep(pauseTime)
		}
		firstTime = false

		spaces, err := modelsOrm.FindPalSpacesAll(o.GetOrmer())
		if err != nil {
			log.Println(err.Error())
			continue
		}

		// Search for "hot" import spaces.
		for _, spaceHot := range spaces {
			if spaceHot.Status == modelsOrm.SPACE_STATUS_TOPROCEED {
				// Start in background mode.
				go o.importSpace(spaceHot) // We do not catch error here.
			}
		}

		// Search for "updating" spaces.
		for _, space := range spaces {
			//log.Println(strconv.Itoa(int(time.Now().Sub(space.ProceededAt))))
			//log.Println(strconv.Itoa(int(int(time.Now().Sub(space.ProceededAt)) / int(time.Minute))))
			if space.Status == modelsOrm.SPACE_STATUS_COMPLITED && time.Now().Sub(space.ProceededAt) > updateRepeatTime {
				o.updateSpace(space) // We do not catch error here.
			}
		}
	}
}

func (o *WorkerMainInstance) importSpace(space *modelsOrm.PalSpace) (err error) {
	log.Println(fmt.Sprintf("Start import space %s", space.Uuid))

	// Mark as started.
	if err = space.UpdateStatus(o.GetOrmer(), modelsOrm.SPACE_STATUS_STARTED, ""); err != nil {
		return o.finishImportUpdate(space, err)
	}

	// Check Library.
	if err = o.makeLibrary(space); err != nil {
		return o.finishImportUpdate(space, err)
	}

	// Proceed importer
	if err = o.getPal().ProceedImport(space, o.GetOrmer()); err != nil {
		return o.finishImportUpdate(space, err)
	}

	return o.finishImportUpdate(space, err)
}

func (o *WorkerMainInstance) updateSpace(space *modelsOrm.PalSpace) (err error) {
	log.Println(fmt.Sprintf("Start update space %s", space.Uuid))

	return o.finishImportUpdate(space, err)
}

func (o *WorkerMainInstance) makeLibrary(space *modelsOrm.PalSpace) (err error) {
	if space.LibraryId == "" {
		// Create new Library.
		var library *modelsHyper.LibraryResponse
		library, err = o.getHyper().CreateLibrary(space.Name, space.CompanyId)
		if err == nil {
			err = space.SetLibraryId(o.GetOrmer(), library.Id)
		}
	}
	return
}

func (o *WorkerMainInstance) getHyper() *Hyper {
	o.onceGetHyper.Do(func () {
		o.hyper = new(Hyper)
	})
	return o.hyper
}

func (o *WorkerMainInstance) getPal() *AssetLibraryPhillips {
	o.onceGetPal.Do(func () {
		o.pal = new(AssetLibraryPhillips)
	})
	return o.pal
}

// Silent mode.
func (o *WorkerMainInstance) updateProceededAt(space *modelsOrm.PalSpace) {
	if err := space.UpdateProceededAt(o.GetOrmer()); err != nil {
		log.Println("Can not update proceeded_at: " + err.Error())
	}
}

// Silent mode.
func (o *WorkerMainInstance) finishImportUpdate(space *modelsOrm.PalSpace, err error) (errBack error) {
	errBack = err
	if err == nil {
		if space.Status != modelsOrm.SPACE_STATUS_COMPLITED {
			err = space.UpdateStatus(o.GetOrmer(), modelsOrm.SPACE_STATUS_COMPLITED, "")
		}
	} else {
		message := err.Error()
		err = nil
		if space.Status != modelsOrm.SPACE_STATUS_COMPLITED {
			space.Status = modelsOrm.SPACE_STATUS_ERROR
		}
		err = space.UpdateStatus(o.GetOrmer(), space.Status, message)
	}

	if err != nil {
		log.Println("Can not update status to finish: " + err.Error())
	}
	o.updateProceededAt(space)
	log.Println(fmt.Sprintf("Finish import/update space %s", space.Uuid))
	return
}