package service

import (
	"github.com/astaxie/beego"
	"sync"
	"log"
	"github.com/astaxie/beego/httplib"
	"strconv"
	"fmt"
	"encoding/json"
	"errors"
	"hyper-pal/models/pal"
	"hyper-pal/models/data"
	"os"
	"io"
	"archive/zip"
	"github.com/satori/go.uuid"
	"net/http"
	"hyper-pal/models/orm"
	"github.com/astaxie/beego/orm"
)

type AssetLibraryPhillips struct {

	onceGetApiUrl 	sync.Once
	apiUrl			string
	mutexToken		sync.Mutex
	token			string
	onceAuthToken	sync.Once
	authToken		string
	onceRegHeader	sync.Once
	regHeader		string
	onceGetHyper	sync.Once
	hyper			*Hyper
}

func (o *AssetLibraryPhillips) ProceedImport(space *modelsOrm.PalSpace, ormer orm.Ormer) (err error) {
	// Prepare all Records for Classification.
	records, err := o.getRecords(space.ClassificationId)
	if err != nil {
		return
	}
	log.Println(fmt.Sprintf("Got %d records from PAL", len(records.Items)))

	for _, record := range records.Items {
		// Prepare File from Record.
		filePal, err := o.proceedRecord(&record)
		if err != nil {
			continue
		}
		file := modelsData.CreateFileFromPal(filePal)

		// Upload to ContentItem.
		contentItemId, err := o.uploadFile(file, space.LibraryId)
		if err != nil {
			continue
		}

		// Save data to DB.
		_, err = ormer.Insert(modelsOrm.CreateFileInPal(contentItemId, space.Uuid, record.Id, file))
		if err != nil {
			log.Println("AHTUNG! Can not insert files_in_pal: " + err.Error())
			continue
		}
		log.Println(fmt.Sprintf("File uploaded (PalFileId / contentItemId): %s / %s", file.ExternalId, contentItemId))

		//text, _ := json.Marshal(file)
		//log.Println(contentItemId, string(text)/*, file.OutFilename, file.Description*/)
		//break
	}

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
	if err := o.CopyFile(filename, filename + "_copy"); err != nil {
		log.Println(err.Error())
		return err
	}
	filename = filename + "_copy"

	// Create container.
	file := modelsData.CreateFileFromPal(&modelsPal.File{
		Id: "9a86f7f6-b22f-4b35-8102-54fbf9c603a5",
		FileName: "testImage.jpg",
		FileSize: 1824194,
		OutFilename: filename,
		Description: "Test description",
	})

	// Upload
	if _, err := o.uploadFile(file, "96ae7ea0-20a5-11e3-a2bc-001ec9b84463"); err != nil {
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

func (o *AssetLibraryPhillips) uploadFile(file *modelsData.File, libraryId string) (contentItemId string, err error) {
	// Do not forget delete temp file.
	defer o.deleteFile(file.Fullpath)

	// Upload in Hyper service.
	contentItemId, err = o.getHyper().UploadFile(file, libraryId)

	return
}

func (o *AssetLibraryPhillips) proceedRecord(record *modelsPal.Record) (*modelsPal.File, error) {
	// Get MasterFile for Record.
	file, err := o.getFile(record.MasterFile.Id)
	if err != nil {
		return nil, err
	}

	// Find description of file infields of Record.
	if fieldDescription := record.Fields.FindByFieldName("Asset_Description"); fieldDescription != nil {
		file.Description = fieldDescription.GetValueFirstString()
	}

	// Order direct link to file.
	link, err := o.getFileDownloadLink(record.Id)
	if err != nil {
		return nil, err
	}

	// Try to open file.
	request, err := o.createGetRequest(link)
	if err != nil {
		return nil, err
	}
	response, err := request.Response()
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	defer response.Body.Close()

	// Download file, save it, unzip.

	// Create zip file in OS.
	zipFilename := o.getTmpPath() + uuid.NewV4().String()
	zipFile, err := os.Create(zipFilename)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	defer zipFile.Close()
	defer o.deleteFile(zipFilename)
	// Copy response to zipfile.
	_, err = io.Copy(zipFile, response.Body)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}

	// Create zipReader from zipfile.
	zipReader, err := zip.OpenReader(zipFilename)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	defer zipReader.Close()
	// Create original file from zip with source filename.
	file.OutFilename = o.getTmpPath() + uuid.NewV4().String()
	outFile, err := os.Create(file.OutFilename)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	defer outFile.Close()
	fileSize := int64(0)
	for _, unzFile := range zipReader.File {
		// Unzip only first file from directory.
		unzFileReader, err := unzFile.Open()
		if err != nil {
			log.Println(err.Error())
			return nil, err
		}
		fileSize, err = io.Copy(outFile, unzFileReader)
		if err != nil {
			log.Println(err.Error())
			return nil, err
		}
		//log.Println(strconv.Itoa(int(fileSize)))
		break // Only first
	}

	// Check filesize correct
	if fileSize != int64(file.FileSize) {
		o.deleteFile(file.OutFilename)
		message := fmt.Sprintf("Unziped filesize is %d, but must be %d", fileSize, file.FileSize)
		return nil, errors.New(message)
	}

	return file, nil
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

func (o *AssetLibraryPhillips) getRecords(classificationId string) (*modelsPal.Records, error) {
	request, err := o.createGetRequest("records")
	if err != nil {
		return nil, err
	}
	o.setRequestPagintaion(request, 1, 10, "createdon")
	o.setRequestFilter(request, "classification=" + classificationId)
	o.setRequestFields(request, "fields,masterfile") // "fields,classifications,masterfile"
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

	if (response.StatusCode < http.StatusOK || response.StatusCode >= 300) {
		log.Println(fmt.Sprintf("Can not get Records (%s): %s", strconv.Itoa(response.StatusCode), body))
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

	if (response.StatusCode < http.StatusOK || response.StatusCode >= 300) {
		log.Println(fmt.Sprintf("Can not get File (%s): %s", strconv.Itoa(response.StatusCode), body))
		return nil, err
	}

	var file *modelsPal.File
	if err := json.Unmarshal([]byte(body), &file); err != nil {
		log.Println(fmt.Sprintf("Can not decode File: %s", err.Error()))
		return nil, err
	}

	return file, nil
}

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

	if (response.StatusCode < http.StatusOK || response.StatusCode >= 300) {
		log.Println(fmt.Sprintf("Can not get File (%s): %s", strconv.Itoa(response.StatusCode), body))
		return "", err
	}

	var orderFile *modelsPal.OrderFile
	if err := json.Unmarshal([]byte(body), &orderFile); err != nil {
		log.Println(fmt.Sprintf("Can not decode Order(file): %s", err.Error()))
		return "", err
	}

	link := ""
	for _, deliveredFile := range orderFile.DeliveredFiles {
		if deliveredFile != "" {
			link = deliveredFile
			break
		}
	}
	if link == "" {
		message := "No delivered file in Order"
		log.Println(message)
		return "", errors.New(message)
	}

	return link, nil
}

func (o *AssetLibraryPhillips) getApiUrl() string {
	o.onceGetApiUrl.Do(func () {
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
	o.onceAuthToken.Do(func () {
		o.authToken = beego.AppConfig.String("pal.api.auth.token")
		if o.authToken == "" {
			log.Panic("No pal.api.auth.token in config")
		}
	})

	request := httplib.Post(o.getApiUrl() + "/auth").
		Header("Authorization", "Basic " + o.authToken).
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
		log.Println(fmt.Sprintf("Can not auth (%s): %s", strconv.Itoa(response.StatusCode), body))
		return "", err
	}

	var tokenObj *modelsPal.AuthToken
	if err := json.Unmarshal([]byte(body), &tokenObj); err != nil {
		log.Println(fmt.Sprintf("Can not decode token: %s", body))
		return "", err
	}
	o.token = tokenObj.Token
	if o.token == "" {
		message := "Token is empty in response"
		log.Println(message)
		return "", errors.New(message)
	}

	return o.token, nil
}

func (o *AssetLibraryPhillips) getRegistrationHeader() string {
	o.onceRegHeader.Do(func () {
		o.regHeader = beego.AppConfig.String("pal.api.header.registration")
		if o.regHeader == "" {
			log.Panic("No pal.api.header.registration in config")
		}
	})
	return o.regHeader
}

func (o *AssetLibraryPhillips) getHyper() *Hyper {
	o.onceGetHyper.Do(func () {
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
	request := httplib.NewBeegoRequest(o.getApiUrl() + "/" + action, method)
	token, err := o.getToken()
	if err != nil {
		return request, err
	}
	return request.
		Header("Authorization", "Token " + token).
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
