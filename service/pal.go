package service

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/httplib"
	"github.com/astaxie/beego/orm"
	"io"
	"log"
	"net/http"
	"os"
	"pal-importer/models/data"
	"pal-importer/models/orm"
	"pal-importer/models/pal"
	"pal-importer/system"
	"path/filepath"
	_ "path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const PAL_RECORDS_PAGESIZE = 20

var ExcludeList [][]string

type AssetLibraryPhillips struct {
	digMaxDuration time.Duration
	digStartTime   time.Time

	onceGetApiUrl sync.Once
	apiUrl        string
	mutexToken    sync.Mutex
	token         string
	onceAuthToken sync.Once
	authToken     string
	onceRegHeader sync.Once
	regHeader     string
	onceGetHyper  sync.Once
	hyper         *Hyper
}

func isFileExcludedFromUpload(filename string) (result bool) {
	var extension = filepath.Ext(filename)
	if ExcludeList == nil {
		return false
	}
	if ExcludeList[0] == nil {
		return false
	}
	for _, s := range ExcludeList[0] {
		if extension == s {
			return true
		}
	}
	return false
}

func (o *AssetLibraryPhillips) ProceedImport(space *modelsOrm.PalSpace, ormer orm.Ormer) (err error) {
	// Prepare all Records for Classification.
	o.digMaxDuration = 10 * time.Minute
	page := 1
	countProceededWithPages := 0
	countProceed := 0
	countProceedUploaded := 0
	countProceedUpdated := 0
	countProceedSkipped := 0
	records, err := o.getRecords(space.ClassificationId, page)
	if err != nil {
		return
	}
	log.Println(fmt.Sprintf("Starting import PAL with %d records", records.TotalCount))

	for { // proceedRecord and proceedTags are making get requests to download from PalSpace to pal-importer
		log.Println(fmt.Sprintf("Proceed page %d, records %d", page, len(records.Items)))
		for _, record := range records.Items {
			countProceed++
			// Prepare File from Record.
			log.Println(fmt.Sprintf("Proceed record: %s", record.Id))
			filePal, err := o.proceedRecord(&record)
			if err != nil {
				continue
			}

			if isFileExcludedFromUpload(filePal.FileName) {
				log.Println(fmt.Sprintf("Excluded from upload: %s", filePal.FileName))
				continue
			}

			// Prepare tags. CI: fa6f574b-3b5a-4afa-ba8e-82ad9f46990b, recId 32cfd6aaf56647b8b93ba8e300a84f8d
			var tagIds []string
			tagIds, err = o.proceedTags(ormer, &record, space.ClassificationId, space.LibraryId)
			if err != nil {
				continue
			}

			// Check file already imported
			// I think it is checking record by record & not file by file
			contentItemId := ""
			fip, err := modelsOrm.FindOneFileInPalByFirstUpload(ormer, space.Uuid, record.Id, space.LibraryId)
			if err != nil {
				log.Println(err.Error())
				continue
			} else if fip == nil {
				// Download file from PAL.
				// Here is where the problem seems to occur.
				if err = o.proceedRecordDownload(&record, filePal); err != nil {
					continue
				}
				// this seems to be where a CMS file (including path) is created from collection
				// of records in PalSpace
				file := modelsData.CreateFileFromPal(filePal)

				// Upload to ContentItem.
				contentItemId, err = o.uploadFile(file, space.LibraryId, false)
				if err != nil {
					continue
				}

				// Save data to DB.
				_, err = ormer.Insert(modelsOrm.CreateFileInPal(contentItemId, space.LibraryId, space.Uuid, record.Id, file))
				if err != nil {
					log.Println("AHTUNG! Can not insert FilesInPal: " + err.Error())
					continue
				}
				log.Println(fmt.Sprintf("File uploaded (PalFileId / contentItemId): %s / %s", file.ExternalId, contentItemId))
				countProceedUploaded++
			} else {
				// File present in Library, check for updating.
				if fip.HadChanged(filePal) {
					log.Println(fmt.Sprintf("File changed (contentItemId: %s)", fip.ContentItemId))

					// Download file from PAL.
					if err = o.proceedRecordDownload(&record, filePal); err != nil {
						continue
					}
					file := modelsData.CreateFileFromPal(filePal)
					file.Id = fip.ContentItemId

					// Upload to ContentItem.
					contentItemId, err = o.uploadFile(file, space.LibraryId, true)
					if err != nil {
						continue
					}

					// Update data in DB.
					contentItemId, err = fip.UpdateByFile(ormer, filePal, contentItemId)
					if err != nil {
						log.Println("AHTUNG! Can not update FilesInPal: " + err.Error())
						continue
					}
					log.Println(fmt.Sprintf("File updated (PalFileId / contentItemId): %s / %s", file.ExternalId, contentItemId))
					countProceedUpdated++
				} else {
					contentItemId = fip.ContentItemId
					log.Println(fmt.Sprintf("File already uploaded (contentItemId: %s)", fip.ContentItemId))
					countProceedSkipped++
				}
			}

			// Update tags for imported/existing contentItem.
			tagIdsStr := ""
			resourceTagIdsStr := ""
			if len(tagIds) > 0 {
				tagIdsStr = strings.Join(tagIds, "|")
			}
			configIsResource, _ := beego.AppConfig.Bool("hyper.importer.tag.isResource")
			if configIsResource {
				resourceTagIdsStr = tagIdsStr
				tagIdsStr = ""
			}
			if _, err = ormer.Raw("update ContentItemsInLibraries set tags_uuid = ?, resource_tags_uuid = ? where content_item_id = ? and library_uuid = ?", tagIdsStr, resourceTagIdsStr, contentItemId, space.LibraryId).Exec(); err != nil {
				log.Println(fmt.Sprintf("AHTUNG!!! Can not update tags for contentItem (%s)", contentItemId))
				// And do no more.
			}

		}
		countProceededWithPages += PAL_RECORDS_PAGESIZE

		// Do we have to procced more pages?
		if countProceededWithPages >= records.TotalCount {
			break // Finish!
		}
		// Prepare new page for cycling.
		page++
		if records, err = o.getRecords(space.ClassificationId, page); err != nil {
			return
		}
	}

	log.Println(fmt.Sprintf("PAL proceeded (New upload / Updated / Skipped / Error) : (%d / %d / %d / %d) from %d records",
		countProceedUploaded,
		countProceedUpdated,
		countProceedSkipped,
		countProceed-countProceedUploaded-countProceedUpdated-countProceedSkipped,
		countProceed))
	err = nil
	return
}

/*func (o *AssetLibraryPhillips) ProceedClassification(classificationId string) error {
	// Prepare all Records for Classification.
	records, err := o.getRecords(classificationId)
	if err != nil {
		return err
	}

	for _, record := range records.Items {
		file, err := o.proceedRecord(&record)
		if err != nil {
			break;//continue
		}
		if err = o.uploadFile(modelsData.CreateFileFromPal(file)); err != nil {
			break;//continue
		}

		text, _ := json.Marshal(file)
		log.Println(string(text), file.OutFilename, file.Description)
		break
	}

	return nil
}*/

func (o *AssetLibraryPhillips) TestUpload() error {
	// Copy test file
	filename := o.getTmpPath() + "test001.jpg"
	if err := o.CopyFile(filename, filename+"_copy"); err != nil {
		log.Println(err.Error())
		return err
	}
	filename = filename + "_copy"

	// Create container.
	file := modelsData.CreateFileFromPal(&modelsPal.File{
		Id:          "9a86f7f6-b22f-4b35-8102-54fbf9c603a5",
		FileName:    "testImage.jpg",
		FileSize:    1824194,
		OutFilename: filename,
		Description: "Test description",
	})

	// Upload
	if _, err := o.uploadFile(file, "96ae7ea0-20a5-11e3-a2bc-001ec9b84463", false); err != nil {
		return err
	}

	return nil
}

// Copy the src file to dst. Any existing file will be overwritten and will not
// copy file attributes.
func (o *AssetLibraryPhillips) CopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Close()
}

func (o *AssetLibraryPhillips) uploadFile(file *modelsData.File, libraryId string, needDeleteBefore bool) (contentItemId string, err error) {
	// Do not forget delete temp file.
	// Is this file created in a path relative to the temp directory?
	defer o.deleteFile(file.Fullpath)

	// Upload in Hyper service.
	contentItemId, err = o.getHyper().UploadFile(file, libraryId, needDeleteBefore)

	return
}

func (o *AssetLibraryPhillips) proceedRecord(record *modelsPal.Record) (file *modelsPal.File, err error) {
	// Get MasterFile for Record.
	file, err = o.getFile(record.MasterFile.Id)
	if err != nil {
		return
	}

	// Find description of file infields of Record.
	// It seems that Pal-Importer is interested in only one field description & one field title
	// according to the Adam REST API
	if fieldDescription := record.Fields.FindByFieldName("Asset_Description"); fieldDescription != nil {
		file.Description = fieldDescription.GetValueFirstString()
	}
	if fieldTitle := record.Fields.FindByFieldName("Asset_Media_Title"); fieldTitle != nil {
		file.Title = fieldTitle.GetValueFirstString()
	}

	return
}

// STOPPED HERE!!!!

func (o *AssetLibraryPhillips) proceedTags(ormer orm.Ormer, record *modelsPal.Record, topClassificationId, libraryUuid string) (tagIds []string, err error) {
	//defer panic("procced tags")
	/*byte, _ := json.Marshal(record.Classfications.Items)
	println()
	println()
	println(string(byte))
	println()
	println()*/
	// mapping classifications to tags?
	tagIds = make([]string, 0)
	for _, recordClassification := range record.Classfications.Items {
		/*println()
		println(recordClassification.Id)
		println()*/
		var classificationInPal *modelsOrm.ClassificationInPal
		// Check maybe present in our DB.
		classificationInPal, err = new(modelsOrm.ClassificationInPal).LoadByClassificationId(ormer, recordClassification.Id)
		if err != nil {
			return
		} else if classificationInPal != nil {
			// Already present
			if classificationInPal.TagId == nil { // classification with no matching Tag???
				// Dead-end classification.
				//log.Println(fmt.Sprintf("  -- Dead classification (%s)", recordClassification.Id))
				// the classification does not map to a tag
				continue
			} else {
				// Working classification.
				log.Println(fmt.Sprintf("  -- Working classification (%s)", recordClassification.Id))
			}
		} else {
			var classification *modelsPal.Classification
			classification, err = o.getClassification(recordClassification.Id)
			if err != nil {
				return
			}
			if classification == nil {
				// No access, save and ignore it.
				classification := &modelsPal.Classification{
					Id: recordClassification.Id,
				}
				if classificationInPal, err = new(modelsOrm.ClassificationInPal).Insert(ormer, classification, nil, topClassificationId, libraryUuid); err != nil {
					return
				}
				continue
			}

			if classificationInPal, err = o.autoGetClassificationsInPal(ormer, classification, topClassificationId, libraryUuid); err != nil {
				return
			}
			if classificationInPal == nil { // keep looking for classification that can be mapped to a Tag???
				// No tags for this classification.
				continue
			}
		}

		// Prepared worked classification.
		var tagIdsOne []string
		if tagIdsOne, err = o.GetTagIds(ormer, classificationInPal, topClassificationId); err != nil {
			return
		}
		for _, tagId := range tagIdsOne {
			// Do not duplicate tags.
			found := false
			for _, tagIdMain := range tagIds {
				if tagId == tagIdMain {
					found = true
					break
				}
			}
			if !found {
				tagIds = append(tagIds, tagId)
			}
		}
	}

	if len(tagIds) > 0 {
		byteTags, err2 := json.Marshal(tagIds)
		if err2 == nil {
			log.Println(fmt.Sprintf("  -- Tags: %s", string(byteTags)))
		} else {
			log.Println("Error: " + err.Error())
		}
	}

	return
}

/**
 * Returns nil if no classification or top classification is different (or it is top) or it is root.
 */
func (o *AssetLibraryPhillips) autoGetClassificationsInPal(ormer orm.Ormer, classification *modelsPal.Classification, topClassificationId, libraryUuid string) (classificationInPal *modelsOrm.ClassificationInPal, err error) {
	log.Println(fmt.Sprintf("  -- Start classification (%s)", classification.Id)) // Debug
	/*if classification.Id == topClassificationId {
		// It is top classification.
		log.Println(fmt.Sprintf("It is top classification (%s)", classification.Id)) // Debug
		return
	}*/
	if !classification.IsRoot && classification.ParentId == "" {
		// It is root classification.
		log.Println(fmt.Sprintf("  -- It is not root classification but without parent (%s)", classification.Id)) // Debug
		return
	}

	// Try to detect classficationInPal.
	if classificationInPal, err = new(modelsOrm.ClassificationInPal).LoadByClassificationId(ormer, classification.Id); err != nil {
		log.Println(fmt.Sprintf("  -- Can not read classification (%s) from DB: %s", classification.Id, err.Error()))
		return
	}
	if classificationInPal != nil {
		// Classification known.
		log.Println(fmt.Sprintf("  -- Classification found (%s)", classification.Id)) // Debug
		return
	}
	// if the Classification is new, how do we map it to a tag???
	// multiple classifications can go to multiple tags, but the logic remains complex.
	// Classification is new, detect parent.
	classificationInPal = new(modelsOrm.ClassificationInPal)
	var parentTagId *string
	if /*classification.ParentId == topClassificationId ||*/ classification.IsRoot {
		// I am very first, add to DB.
		parentTagId = nil
	} else {
		// Read parent classification from PAL.
		var parentClassification *modelsPal.Classification
		parentClassification, err = o.getClassification(classification.ParentId)
		if err != nil || parentClassification == nil {
			return
		}

		var parentClassificationInPal *modelsOrm.ClassificationInPal
		if parentClassificationInPal, err = o.autoGetClassificationsInPal(ormer, parentClassification, topClassificationId, libraryUuid); err != nil || parentClassificationInPal == nil {
			return
		}
		parentTagId = parentClassificationInPal.TagId // is this where we create a new Tag???
		// we gave extract classificationinPal to TagsinLibraries
	}

	// Add to DB
	if _, err = classificationInPal.Insert(ormer, classification, parentTagId, topClassificationId, libraryUuid); err != nil {
		log.Println(fmt.Sprintf("  -- Can not insert classificationInPal (%s) in DB: %s", classification.Id, err.Error()))
		return
	}
	if classificationInPal.TagId == nil {
		log.Println(fmt.Sprintf("  -- Classification (%s) added to DB with EMPTY tag", classificationInPal.ClassificationId))
	} else {
		log.Println(fmt.Sprintf("  -- Classification (%s) added to DB with tag (%s)", classificationInPal.ClassificationId, *classificationInPal.TagId))
	}

	return
}

/**
 * Returns empty slice if classifications are not part of topClassificationId.
 */
// How is the array of tagIds used????
func (o *AssetLibraryPhillips) GetTagIds(ormer orm.Ormer, classificationInPal *modelsOrm.ClassificationInPal, topClassificationId string) (tagIds []string, err error) {
	tagIds = make([]string, 0)
	if classificationInPal.ClassificationId == topClassificationId || classificationInPal.ParentClassificationId == nil {
		// Top classification, return nothing.
		return
	}
	if classificationInPal.TagId == nil {
		// Child has no tag, means all tree does not have.
		return
	}
	// Read all classificationsInPal tree for every child.
	// Are we walking up the Classifications?
	allCips := make([]*modelsOrm.ClassificationInPal, 0)
	allCips = append(allCips, classificationInPal)
	tagIds = append(tagIds, *classificationInPal.TagId)
	cipParentId := classificationInPal.ParentClassificationId
	for cipParentId != nil {
		cip := new(modelsOrm.ClassificationInPal)
		if _, err = cip.LoadByClassificationId(ormer, *cipParentId); err != nil {
			log.Println(fmt.Sprintf("  -- Can not load classificationInPal (%s) from DB: %s", *cipParentId, err.Error()))
			return
		}
		if cip.TagId != nil {
			// Only if classificationInPal has tag.
			allCips = append(allCips, cip)
			tagIds = append(tagIds, *cip.TagId)
		}
		cipParentId = cip.ParentClassificationId
	}

	/*println()
	for _, cip := range allCips {
		println(*cip.TagId)
	}
	println()*/
	/*println()
	for _, tag := range tagIds {
		println(tag)
	}
	println()*/

	return
}

func (o *AssetLibraryPhillips) proceedRecordDownload(record *modelsPal.Record, file *modelsPal.File) (err error) {
	// Order direct link to file.
	link, err := o.getFileDownloadLink(record.Id)
	if err != nil {
		return
	}

	// Try to open file.
	request, err := o.createGetRequest(link)
	if err != nil {
		return
	}
	request.SetTimeout(10*time.Second, 10*time.Minute)
	response, err := request.Response()
	if err != nil {
		log.Println(err.Error())
		return
	}
	defer response.Body.Close()

	// Download file, save it, unzip.

	// Create zip file in OS.
	// so far I have only found this proceedure for zipped files.
	zipFilename := o.getTmpPath() + system.NewV4String()
	zipFile, err := os.Create(zipFilename)
	if zipFile != nil {
		defer o.closeRemoveZip(zipFile, zipFilename)
	}
	if err != nil {
		log.Println(err.Error())
		return
	}
	// Copy response to zipfile.
	fileSize, err := io.Copy(zipFile, response.Body)
	if err != nil {
		log.Println(err.Error())
		return
	}

	if response.Header.Get("Content-Type") == "application/zip" {
		// Create zipReader from zipfile.
		var zipReader *zip.ReadCloser
		zipReader, err = zip.OpenReader(zipFilename)
		if err != nil {
			log.Println(fmt.Sprintf("Open zip (%s / %s):"+err.Error(), record.MasterFile.Id, record.Id))
			return
		}
		defer zipReader.Close()
		// Create original file from zip with source filename.
		file.OutFilename = o.getTmpPath() + system.NewV4String()
		var outFile *os.File
		outFile, err = os.Create(file.OutFilename)
		if err != nil {
			log.Println(err.Error())
			return
		}
		defer outFile.Close()
		for _, unzFile := range zipReader.File {
			// Unzip only first file from directory.
			var unzFileReader io.ReadCloser
			unzFileReader, err = unzFile.Open()
			if err != nil {
				log.Println(fmt.Sprintf("Open zipfile in arcihve (%s):"+err.Error(), record.MasterFile.Id))
				return
			}
			fileSize, err = io.Copy(outFile, unzFileReader)
			if err != nil {
				log.Println(fmt.Sprintf("Save unzipped file (%s):"+err.Error(), record.MasterFile.Id))
				return
			}
			//log.Println(strconv.Itoa(int(fileSize)))
			break // Only first
		}
	} else {
		// Just use original file.
		// What if original file was not zipped.
		file.OutFilename = zipFilename
	}
	// *** IS THE LOGIC HERE CORRECT FOR NON-ZIPPED PAL FILE -- LOOKS OKAY BUT MIGHT NOT BE!!
	// Check filesize correct
	if fileSize != int64(file.FileSize) {
		o.deleteFile(file.OutFilename)
		err = errors.New(fmt.Sprintf("Unziped (downloaded) filesize is %d, but must be %d", fileSize, file.FileSize))
		log.Println(err.Error())
		return
	}

	return
}

func (o *AssetLibraryPhillips) closeRemoveZip(file *os.File, filename string) {
	file.Close()
	o.deleteFile(filename) // here we get rid of Zip file.

}

func (o *AssetLibraryPhillips) deleteFile(filename string) error {
	err := os.Remove(filename)
	if err != nil {
		log.Println(err.Error())
	}
	return err
}

func (o *AssetLibraryPhillips) getTmpPath() string {
	return beego.AppPath + "/tmp/"
}

// getting records associated with a classifications???
func (o *AssetLibraryPhillips) getRecords(classificationId string, page int) (*modelsPal.Records, error) {
	request, err := o.createGetRequest("records")
	if err != nil {
		return nil, err
	}
	o.setRequestPagintaion(request, page, PAL_RECORDS_PAGESIZE, "createdon")
	o.setRequestFilter(request, "classification="+classificationId)
	o.setRequestFields(request, "fields,classifications,masterfile")
	response, err := request.Response()
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	defer response.Body.Close()
	body, err := request.String()
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= 300 {
		err = errors.New(fmt.Sprintf("Can not get Records (%s): %s", strconv.Itoa(response.StatusCode), body))
		log.Println(err.Error())
		return nil, err
	}

	var records *modelsPal.Records
	if err := json.Unmarshal([]byte(body), &records); err != nil {
		log.Println(fmt.Sprintf("Can not decode Records: %s", err.Error()))
		return nil, err
	}

	//text, _ := json.Marshal(records)
	//log.Println(string(text))
	return records, nil
}

//masterfileID applies to a group of file versions???
func (o *AssetLibraryPhillips) getFile(masterfileId string) (*modelsPal.File, error) {
	request, err := o.createGetRequest("file/" + masterfileId + "/latestversion")
	if err != nil {
		return nil, err
	}
	response, err := request.Response()
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	defer response.Body.Close()
	body, err := request.String()
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= 300 {
		err = errors.New(fmt.Sprintf("Can not get File (%s): %s", strconv.Itoa(response.StatusCode), body))
		log.Println(err.Error())
		return nil, err
	}

	var file *modelsPal.File
	if err := json.Unmarshal([]byte(body), &file); err != nil {
		log.Println(fmt.Sprintf("Can not decode File: %s", err.Error()))
		return nil, err
	}

	return file, nil
}

/**
 * Returns nil if 404 (PAL shows unaccessable classifications with 404).
 */

// checking a classification for sanity???
func (o *AssetLibraryPhillips) getClassification(classificationId string) (classification *modelsPal.Classification, err error) {
	request, err := o.createGetRequest("classification/" + classificationId)
	if err != nil {
		return
	}
	response, err := request.Response()
	if err != nil {
		log.Println(err.Error())
		return
	}
	defer response.Body.Close()
	body, err := request.String()
	if err != nil {
		log.Println(err.Error())
		return
	}

	if response.StatusCode == http.StatusNotFound {
		message := fmt.Sprintf("  -- No access to Classification (%s)", classificationId) // Not error!
		log.Println(message)
		classification = nil
		return
	} else if response.StatusCode < http.StatusOK || response.StatusCode >= 300 {
		err = errors.New(fmt.Sprintf("Can not get Classification (%s): %s", strconv.Itoa(response.StatusCode), body))
		log.Println(err.Error())
		return
	}

	classification = new(modelsPal.Classification)
	if err = json.Unmarshal([]byte(body), &classification); err != nil {
		log.Println(fmt.Sprintf("Can not decode Classification: %s", err.Error()))
		return
	}

	return
}

// is this really a file downloadlink or is it a record download link???
// may all records that come from a file are associated with a specific file download link
func (o *AssetLibraryPhillips) getFileDownloadLink(recordId string) (string, error) {
	bodyRequest, err := json.Marshal(modelsPal.CreateBodyFileOrder().AddTargetRecordId(recordId))
	//log.Println(string(bodyRequest))
	if err != nil {
		return "", err
	}
	request, err := o.createPostRequest("orders", bodyRequest)
	if err != nil {
		return "", err
	}
	response, err := request.Response()
	if err != nil {
		log.Println(err.Error())
		return "", err
	}
	defer response.Body.Close()
	body, err := request.String()
	if err != nil {
		log.Println(err.Error())
		return "", err
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= 300 {
		err = errors.New(fmt.Sprintf("Can not Order file (%s): %s", strconv.Itoa(response.StatusCode), body))
		log.Println(err.Error())
		return "", err
	}

	var orderFile *modelsPal.OrderFile
	if err := json.Unmarshal([]byte(body), &orderFile); err != nil {
		log.Println(fmt.Sprintf("Can not decode Order(file): %s", err.Error()))
		return "", err
	}

	o.digStartTime = time.Now()
	return o.digLinkFromOrder(orderFile)
}

// Can files be ordered from PAL????
func (o *AssetLibraryPhillips) digLinkFromOrder(order *modelsPal.OrderFile) (link string, err error) {
	if order.Status == modelsPal.ORDER_STATUS_SUCCESS {
		link = order.GetFirstFileLink()
		if link == "" {
			err = errors.New(fmt.Sprintf("No link in Success ordered file,, order %s", order.Id))
		}
	} else if order.Status == modelsPal.ORDER_STATUS_EXECUTING {
		// Executing.
		if time.Now().Sub(o.digStartTime) > o.digMaxDuration {
			err = errors.New(fmt.Sprintf("Too much time for executing order, aborting, order %s", order.Id))
		} else {
			//log.Println(fmt.Sprintf("Waiting order %s for executing (%s)...", order.Id, order.ExecutionTime))
			log.Println(fmt.Sprintf("Waiting order %s for executing...", order.Id))
			time.Sleep(1 * time.Second)
			// Ask for new order object.
			request, err := o.createGetRequest("order/" + order.Id)
			if err != nil {
				return "", err
			}
			response, err := request.Response()
			if err != nil {
				log.Println(err.Error())
				return "", err
			}
			defer response.Body.Close()
			body, err := request.String()
			if err != nil {
				log.Println(err.Error())
				return "", err
			}
			if response.StatusCode < http.StatusOK || response.StatusCode >= 300 {
				err = errors.New(fmt.Sprintf("Can not get Order (%s): %s", strconv.Itoa(response.StatusCode), body))
				log.Println(err.Error())
				return "", err
			}
			var orderFile *modelsPal.OrderFile
			if err := json.Unmarshal([]byte(body), &orderFile); err != nil {
				log.Println(fmt.Sprintf("Can not decode Order: %s", err.Error()))
				return "", err
			}
			return o.digLinkFromOrder(orderFile)
		}
	} else if order.Status == "" {
		err = errors.New(fmt.Sprintf("Empty status in order %s", order.Id))
	} else {
		err = errors.New(fmt.Sprintf("Undefined status: \"%s\" in order", order.Status, order.Id))
	}

	if err != nil {
		log.Println(err.Error())
	}
	return
}

// pal-importer being told to use a specific URI???
func (o *AssetLibraryPhillips) getApiUrl() string {
	o.onceGetApiUrl.Do(func() {
		o.apiUrl = beego.AppConfig.String("pal.api.url")
		if o.apiUrl == "" {
			log.Panic("No pal.api.url in config")
		}
	})
	return o.apiUrl
}

func (o *AssetLibraryPhillips) getToken() (string, error) {
	o.mutexToken.Lock()
	defer o.mutexToken.Unlock()
	if o.token == "" {
		if _, err := o.refreshToken(); err != nil {
			return "", err
		}
	}
	return o.token, nil
}

func (o *AssetLibraryPhillips) refreshToken() (string, error) {
	o.onceAuthToken.Do(func() {
		o.authToken = beego.AppConfig.String("pal.api.auth.token")
		if o.authToken == "" {
			log.Panic("No pal.api.auth.token in config")
		}
	})

	request := httplib.Post(o.getApiUrl()+"/auth").
		Header("Authorization", "Basic "+o.authToken).
		Header("Registration", o.getRegistrationHeader()).
		Header("api-version", "1")
	response, err := request.Response()
	if err != nil {
		log.Println(err.Error())
		return "", err
	}
	defer response.Body.Close()
	body, err := request.String()
	if err != nil {
		log.Println(err.Error())
		return "", err
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= 300 {
		err = errors.New(fmt.Sprintf("Can not auth (%s): %s", strconv.Itoa(response.StatusCode), body))
		log.Println(err.Error())
		return "", err
	}

	var tokenObj *modelsPal.AuthToken
	if err := json.Unmarshal([]byte(body), &tokenObj); err != nil {
		log.Println(fmt.Sprintf("Can not decode token: %s", body))
		return "", err
	}
	o.token = tokenObj.Token
	if o.token == "" {
		err = errors.New("Token is empty in response")
		log.Println(err.Error())
		return "", err
	}

	return o.token, nil
}

func (o *AssetLibraryPhillips) getRegistrationHeader() string {
	o.onceRegHeader.Do(func() {
		o.regHeader = beego.AppConfig.String("pal.api.header.registration")
		if o.regHeader == "" {
			log.Panic("No pal.api.header.registration in config")
		}
	})
	return o.regHeader
}

func (o *AssetLibraryPhillips) getHyper() *Hyper {
	o.onceGetHyper.Do(func() {
		o.hyper = new(Hyper)
	})
	return o.hyper
}

func (o *AssetLibraryPhillips) createGetRequest(action string) (*httplib.BeegoHTTPRequest, error) {
	return o.createHttpRequest(action, "GET")
}

// Bod supports string and []byte.
func (o *AssetLibraryPhillips) createPostRequest(action string, body interface{}) (*httplib.BeegoHTTPRequest, error) {
	request, err := o.createHttpRequest(action, "POST")
	if err == nil {
		request.Body(body)
	}
	return request, err
}

func (o *AssetLibraryPhillips) createHttpRequest(action, method string) (*httplib.BeegoHTTPRequest, error) {
	request := httplib.NewBeegoRequest(o.getApiUrl()+"/"+action, method)
	token, err := o.getToken()
	if err != nil {
		return request, err
	}
	return request.
			Header("Authorization", "Token "+token).
			Header("Registration", o.getRegistrationHeader()).
			Header("api-version", "1").
			Header("accept-encoding", "gzip, deflate").
			Header("accept-language", "en-US,en;q=0.8").
			Header("Content-Type", "application/hal+json").
			Header("user-agent", "Mozilla/5.0 (Windows NT 6.3; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/60.0.3112.113 Safari/537.36").
			Header("accept", "application/json"),
		nil
}

func (o *AssetLibraryPhillips) setRequestPagintaion(request *httplib.BeegoHTTPRequest, page, pagesize int, sort string) *httplib.BeegoHTTPRequest {
	return request.
		Header("page", strconv.Itoa(page)).
		Header("pagesize", strconv.Itoa(pagesize)).
		Header("sort", sort)
}

func (o *AssetLibraryPhillips) setRequestFilter(request *httplib.BeegoHTTPRequest, filter string) *httplib.BeegoHTTPRequest {
	return request.Header("filter", filter)
}

func (o *AssetLibraryPhillips) setRequestFields(request *httplib.BeegoHTTPRequest, fields string) *httplib.BeegoHTTPRequest {
	return request.Header("select-record", fields)
}
