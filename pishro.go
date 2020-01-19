package pishrocloud

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// Storage ...
type Storage struct {
	APIKey           string
	AuthURL          string
	SwiftURL         string
	UserName         string
	PassWord         string
	DefaultContainer string // todo: add default container
}

// Object ...
type Object struct {
	ContentType string
	ObjectID    string
}

var NilObject Object

// MakeRequest ...
func MakeRequest(Method string, URL string, Token string, Headers map[string]string, Object io.Reader) http.Response {
	client := http.Client{
		Timeout: time.Second * 120,
	}
	request, err := http.NewRequest(Method, URL, Object)

	if err != nil {
		log.Fatalln(err)
	}

	// Default Headers
	if Token != "" {
		request.Header.Set("X-Auth-Token", Token)
	}
	// request.Header.Set("Content-type", "application/json")
	request.Header.Set("Accept", "application/json")

	if Headers != nil {
		for key, value := range Headers {
			request.Header.Set(key, value)
			// println(key + ":" + value)
		}
	}

	resp, err := client.Do(request)

	if err != nil {
		log.Fatalln(err)
	}

	return *resp
}

/*
	Auth API Functioins
*/

// RefreshToken ...
// by default, token exist for 24 hours and you should refresh it every 24h on your program
func (p *Storage) RefreshToken() bool {
	var bodyJSON = []byte(
		fmt.Sprintf("{\"auth\":{\"identity\":{\"methods\":[\"password\"],\"password\":{\"user\":{\"name\":\"%s\",\"domain\":{\"name\":\"default\"},\"password\":\"%s\"}}}}}",
			p.UserName,
			p.PassWord,
		),
	)

	body := bytes.NewReader(bodyJSON)
	var response = MakeRequest("POST", p.AuthURL, "", nil, body)
	defer response.Body.Close()
	statusCode := response.StatusCode

	if statusCode != 201 {
		return false
	}

	p.APIKey = response.Header.Get("X-Subject-Token")
	return true
}

/*
	Containers API Functioins
	More Datail: (https://docs.openstack.org/api-ref/object-store/?expanded=#containers)
*/

// CreateContainer ...
func (p *Storage) CreateContainer(name string) bool {
	var response = MakeRequest("PUT", p.SwiftURL+name, p.APIKey, nil, nil)
	statusCode := response.StatusCode
	defer response.Body.Close()
	if statusCode != 201 {
		return false
	}
	return true
}

// DeleteContainer ...
func (p *Storage) DeleteContainer(name string) bool {
	var response = MakeRequest("DELETE", p.SwiftURL+name, p.APIKey, nil, nil)
	statusCode := response.StatusCode
	defer response.Body.Close()
	if statusCode != 204 {
		return false
	}
	return true
}

/*
	Objects API Functioins
	More Datail: (https://docs.openstack.org/api-ref/object-store/?expanded=#objects)
*/

// UploadObject ...
func (p *Storage) UploadObject(path string, fileName string, container string, headers map[string]string) bool {
	object, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	var response = MakeRequest("PUT", p.SwiftURL+container+"/"+fileName, p.APIKey, headers, object)
	defer object.Close()
	statusCode := response.StatusCode
	if statusCode != 201 {
		return false
	}
	return true
}

// DownloadObject ...
func (p *Storage) DownloadObject(path string, fileName string, container string) bool {
	var response = MakeRequest("GET", p.SwiftURL+container+"/"+fileName, p.APIKey, nil, nil)
	out, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
		return false
	}
	defer out.Close()

	_, err = io.Copy(out, response.Body)
	if err != nil {
		log.Fatal(err)
		return false
	}
	return true
}

// IsObjectExist ...
func (p *Storage) IsObjectExist(fileName string, container string) (Object, bool) {
	var response = MakeRequest("HEAD", p.SwiftURL+container+"/"+fileName, p.APIKey, nil, nil)
	statusCode := response.StatusCode

	if statusCode != 200 {
		return NilObject, false
	}

	return Object{
		ContentType: response.Header.Get("Content-Type"),
	}, true
}
