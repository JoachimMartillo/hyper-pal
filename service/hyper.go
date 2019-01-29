package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/httplib"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"pal-importer/models/data"
	"pal-importer/models/hyper"
	"strconv"
	"sync"
)

type Hyper struct {
	onceGetApiUrl            sync.Once
	apiUrl                   string
	onceGetUserEmail         sync.Once
	userEmail                string
	onceGetAuthEmailPassword sync.Once
	authEmail                string
	authPassword             string
	onceGetLanguage          sync.Once
	languageUuid             string
	languageName             string
	languageIdentifier       string
}

// These routines are for uploading from the pal-importer to hyper-cms.

func (o *Hyper) UploadFile(file *modelsData.File, libraryId string, isUpdating bool) (contentItemId string, err error) {
	// Prepare auth
	cookies, err := o.sendAuthSession()
	if err != nil {
		return
	}

	/*if needDeleteBefore && file.Id != "" {
		o.deleteContentItem(file.Id, libraryId, cookies) // Ignore deleting error.
	}*/

	// Upload file to Hyper temporary.
	log.Println("Trying uploading file...")
	body, err := Upload(o.getApiUrl()+"uploadFile", file.Fullpath, cookies)
	if err != nil {
		log.Println(err.Error())
		return
	}
	// Parse temp filename.
	var fileData map[string]string
	err = json.Unmarshal(body, &fileData)
	if err != nil {
		log.Println(err.Error())
		return
	}
	tmpFilename := ""
	for _, filename := range fileData {
		tmpFilename = filename
		break
	}
	if tmpFilename == "" {
		err = errors.New("No tmp filename in Upload request")
		log.Println(err.Error())
		return
	}

	// Create contentItem
	// making or updating a database record
	contentItem := modelsHyper.CreateContentItemFromFile(file)
	contentItem.SubmittedBy = o.getUserEmail()
	contentItem.TemporaryFilePath = tmpFilename
	//txt, _ := json.Marshal(contentItem)
	//log.Println(string(txt))

	// Push contentItem.
	if isUpdating {
		log.Println(fmt.Sprintf("Trying updating contentItem %s...", file.Id))
		_, err = o.updateContentItem(contentItem, libraryId, cookies)
		contentItemId = contentItem.Id
	} else {
		log.Println("Trying creating contentItem...")
		contentItemId, err = o.createContentItem(contentItem, libraryId, cookies)
	}
	if err != nil {
		return
	}
	//log.Println(string(bodyId))

	return
}

func Upload(url, file string, cookies []*http.Cookie) (body []byte, err error) {
	// Prepare a form that you will submit to that URL.
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	// Add your image file
	f, err := os.Open(file)
	if err != nil {
		return
	}
	defer f.Close()
	fw, err := w.CreateFormFile("image", file)
	if err != nil {
		return
	}
	if _, err = io.Copy(fw, f); err != nil {
		return
	}
	// Add the other fields
	if fw, err = w.CreateFormField("key"); err != nil {
		return
	}
	if _, err = fw.Write([]byte("KEY")); err != nil {
		return
	}
	// Don't forget to close the multipart writer.
	// If you don't close it, your request will be missing the terminating boundary.
	w.Close()

	// Now that you have a form, you can submit it to your handler.
	req, err := http.NewRequest("POST", url, &b)
	if err != nil {
		return
	}
	// Don't forget to set the content type, this will contain the boundary.
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	// Submit the request
	httpClient := &http.Client{}
	res, err := httpClient.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()

	// Check the response
	if res.StatusCode != http.StatusOK {
		err = fmt.Errorf("bad status: %s", res.Status)
		return
	}

	// Read response.
	body, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}

	return
}

// I think a new PAL was found and an hyperCMS library needed to be created.

func (o *Hyper) CreateLibrary(title, companyId string) (libraryResponse *modelsHyper.LibraryResponse, err error) {
	cookies, err := o.sendAuthSession()
	if err != nil {
		return
	}

	languageUuid, languageIdentifier, languageName := o.getLanguage()
	requestBody, err := json.Marshal(modelsHyper.CreateLibraryRequest(title, languageUuid, languageIdentifier, languageName))
	if err != nil {
		log.Println(err.Error())
		return
	}

	request := httplib.Post(o.getApiUrl()+"company/"+companyId+"/libraries?languageUuid="+languageUuid).
		Body(requestBody).
		Header("Accept", "application/json").
		Header("Content-Type", "application/json")
	for _, cookie := range cookies {
		request.SetCookie(cookie)
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

	if response.StatusCode < http.StatusOK || response.StatusCode >= 300 {
		err = errors.New(fmt.Sprintf("Can not create Library (%s): %s", strconv.Itoa(response.StatusCode), body))
		log.Println(err.Error())
		return
	}

	if err = json.Unmarshal([]byte(body), &libraryResponse); err != nil {
		err = errors.New(fmt.Sprintf("Can not decode LibraryResponse: %s", body))
		log.Println(err.Error())
		return
	}

	return
}

func (o *Hyper) sendAuthSession() (cookies []*http.Cookie, err error) {
	authBody, err := json.Marshal(modelsHyper.CreateAuth(o.getAuthEmailPassword()))
	if err != nil {
		log.Println(err.Error())
		return
	}
	req, err := http.NewRequest(
		"POST",
		o.getApiUrl()+"authenticate",
		bytes.NewBuffer(authBody))
	if err != nil {
		log.Println(err.Error())
		return
	}
	req.Header.Add("Content-Type", "application/json")

	httpClient := new(http.Client)
	doRequest := func(req *http.Request) (resp *http.Response, err error) {
		//log.Printf("%v %v ...", req.Method, req.URL)
		//start := time.Now()
		resp, err = httpClient.Do(req)
		if err != nil {
			log.Println(err.Error())
			return
		}
		//log.Printf("%v %v took %v [%v]", req.Method, req.URL, time.Since(start), resp.Status)
		return
	}

	resp, err := doRequest(req)
	if err != nil {
		log.Println(err.Error())
		return
	}
	defer req.Body.Close()
	bodyb, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err.Error())
		return
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= 300 {
		message := fmt.Sprintf("Auth (%d): %s", resp.StatusCode, string(bodyb))
		log.Println(message)
		err = errors.New(message)
	}

	//log.Println(string(bodyb))
	cookies = resp.Cookies()
	return
}

func (o *Hyper) createContentItem(contentItem *modelsHyper.ContentItem, libraryId string, cookies []*http.Cookie) (body string, err error) {
	requestBody, err := json.Marshal(&contentItem)
	if err != nil {
		log.Println(err.Error())
		return
	}
	//log.Println(string(requestBody))
	request := httplib.Post(o.getApiUrl()+"libraries/"+libraryId+"/contentItems").
		Body(requestBody).
		Header("Accept", "application/json").
		Header("Content-Type", "application/json")
	for _, cookie := range cookies {
		request.SetCookie(cookie)
	}

	response, err := request.Response()
	if err != nil {
		log.Println(err.Error())
		return
	}
	defer response.Body.Close()
	body, err = request.String()
	if err != nil {
		log.Println(err.Error())
		return
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= 300 {
		err = errors.New(fmt.Sprintf("Can not create contentItem (%d): %s", response.StatusCode, body))
		log.Println(err.Error())
		return
	}

	return
}

// tell hypercms about the new ContentItem and into which library it should be put.

func (o *Hyper) updateContentItem(contentItem *modelsHyper.ContentItem, libraryId string, cookies []*http.Cookie) (body string, err error) {
	requestBody, err := json.Marshal(&contentItem)
	if err != nil {
		log.Println(err.Error())
		return
	}
	//log.Println(string(requestBody))
	request := httplib.Put(o.getApiUrl()+"contentItems/"+contentItem.Id+"?dbUuid="+libraryId).
		Body(requestBody).
		Header("Accept", "application/json").
		Header("Content-Type", "application/json")
	for _, cookie := range cookies {
		request.SetCookie(cookie)
	}

	response, err := request.Response()
	if err != nil {
		log.Println(err.Error())
		return
	}
	defer response.Body.Close()
	body, err = request.String()
	if err != nil {
		log.Println(err.Error())
		return
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= 300 {
		err = errors.New(fmt.Sprintf("Can not update contentItem (%d): %s", response.StatusCode, body))
		log.Println(err.Error())
		return
	}

	return
}

func (o *Hyper) deleteContentItem(contentItemId, libraryId string, cookies []*http.Cookie) (err error) {
	request := httplib.Post(o.getApiUrl()+"contentItems/"+contentItemId+"?dbUuid="+libraryId).
		Header("Accept", "application/json").
		Header("Content-Type", "application/json")
	for _, cookie := range cookies {
		request.SetCookie(cookie)
	}

	response, err := request.Response()
	if err != nil {
		log.Println(err.Error())
		return
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= 300 {
		var body string
		body, err = request.String()
		if err != nil {
			log.Println(err.Error())
			return
		}
		err = errors.New(fmt.Sprintf("Can not delete contentItem (%d): %s", response.StatusCode, body))
		log.Println(err.Error())
		return
	}

	return
}

func (o *Hyper) getApiUrl() string {
	o.onceGetApiUrl.Do(func() {
		o.apiUrl = beego.AppConfig.String("hyper.api.url")
		if o.apiUrl == "" {
			log.Panic("No hyper.api.url in config")
		}
	})
	return o.apiUrl
}

func (o *Hyper) getUserEmail() string {
	o.onceGetUserEmail.Do(func() {
		o.userEmail = beego.AppConfig.String("hyper.importer.email")
		if o.userEmail == "" {
			log.Panic("No hyper.importer.email in config")
		}
	})
	return o.userEmail
}

func (o *Hyper) getAuthEmailPassword() (string, string) {
	o.onceGetAuthEmailPassword.Do(func() {
		o.authEmail = beego.AppConfig.String("hyper.api.auth.email")
		if o.authEmail == "" {
			log.Panic("No hyper.api.auth.email in config")
		}
		o.authPassword = beego.AppConfig.String("hyper.api.auth.password")
		if o.authPassword == "" {
			log.Panic("No hyper.api.auth.password in config")
		}
	})
	return o.authEmail, o.authPassword
}

func (o *Hyper) getLanguage() (string, string, string) {
	o.onceGetLanguage.Do(func() {
		o.languageUuid = beego.AppConfig.String("hyper.importer.language.uuid")
		if o.languageUuid == "" {
			log.Panic("No hyper.importer.language.uuid in config")
		}
		o.languageIdentifier = beego.AppConfig.String("hyper.importer.language.identifier")
		if o.languageIdentifier == "" {
			log.Panic("No hyper.importer.language.identifier in config")
		}
		o.languageName = beego.AppConfig.String("hyper.importer.language.name")
		if o.languageName == "" {
			log.Panic("No hyper.importer.language.name in config")
		}
	})
	return o.languageUuid, o.languageIdentifier, o.languageName
}
