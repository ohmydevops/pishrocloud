package pishrocloud

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

// Storage ...
type Storage struct {
	APIKey           string
	AuthURL          string
	SwiftURL         string
	UserName         string
	PassWord         string
	DefaultContainer string
	Retries          int
	ConnectTimeout   time.Duration // TODO: add this.
	ExpireTime       time.Time     // TODO: add this.
	AuthLock         sync.Mutex    // TODO: add this.
	HTTPClient       *http.Client  // TODO: add this.
}

// Object ...
type Object struct {
	ContentType string
	ObjectID    string
}

// NilObject ...
var NilObject Object

// MakeRequest ...
func (p *Storage) MakeRequest(Method string, URL string, Token string, Headers map[string]string, Object io.Reader) http.Response {
	request, err := http.NewRequest(Method, URL, Object)

	if err != nil {
		log.Fatalln(err)
	}

	diff := subtractTime(time.Now().Local(), p.ExpireTime)
	if diff <= 10 {
		p.RefreshToken()
	}
	// Default Headers
	request.Header.Set("X-Auth-Token", p.APIKey)
	// if p.APIKey != "" {
	// 	request.Header.Set("X-Auth-Token", p.APIKey)
	// }
	request.Header.Set("Accept", "application/json")

	if Headers != nil {
		for key, value := range Headers {
			request.Header.Set(key, value)
			// println(key + ":" + value)
		}
	}

	resp, err := p.HTTPClient.Do(request)

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
	var PayloadJSON = []byte(
		fmt.Sprintf("{\"auth\":{\"identity\":{\"methods\":[\"password\"],\"password\":{\"user\":{\"name\":\"%s\",\"domain\":{\"name\":\"default\"},\"password\":\"%s\"}}}}}",
			p.UserName,
			p.PassWord,
		),
	)

	body := bytes.NewReader(PayloadJSON)

	// Lock all requests to refresh token!
	p.AuthLock.Lock()

	var response = p.MakeRequest("POST", p.AuthURL, "", nil, body)
	defer response.Body.Close()
	statusCode := response.StatusCode
	if statusCode != 201 {
		return false
	}
	p.APIKey = response.Header.Get("X-Subject-Token")
	p.ExpireTime = time.Now().Local().Add(time.Second * 30)
	log.Println("refreshed token now: " + p.APIKey)
	p.AuthLock.Unlock()
	// Unlock all requests after refresh token!

	return true
}

/*
	Containers API Functioins
	More Datail: (https://docs.openstack.org/api-ref/object-store/?expanded=#containers)
*/

// CreateContainer ...
func (p *Storage) CreateContainer(name string) bool {
	var response = p.MakeRequest("PUT", p.SwiftURL+name, p.APIKey, nil, nil)
	statusCode := response.StatusCode
	defer response.Body.Close()
	if statusCode != 201 {
		return false
	}
	return true
}

// DeleteContainer ...
func (p *Storage) DeleteContainer(name string) bool {
	var response = p.MakeRequest("DELETE", p.SwiftURL+name, p.APIKey, nil, nil)
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
	var response = p.MakeRequest("PUT", p.SwiftURL+container+"/"+fileName, p.APIKey, headers, object)
	defer object.Close()
	statusCode := response.StatusCode
	if statusCode != 201 {
		return false
	}
	return true
}

// DownloadObject ...
func (p *Storage) DownloadObject(path string, fileName string, container string) bool {
	var response = p.MakeRequest("GET", p.SwiftURL+container+"/"+fileName, p.APIKey, nil, nil)
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
	var response = p.MakeRequest("HEAD", p.SwiftURL+container+"/"+fileName, p.APIKey, nil, nil)
	statusCode := response.StatusCode

	if statusCode != 200 {
		return NilObject, false
	}

	return Object{
		ContentType: response.Header.Get("Content-Type"),
		ObjectID:    fileName,
	}, true
}

// DeleteObject ...
func (p *Storage) DeleteObject(fileName string, container string) bool {
	var response = p.MakeRequest("DELETE", p.SwiftURL+container+"/"+fileName, p.APIKey, nil, nil)
	statusCode := response.StatusCode

	if statusCode != 204 {
		return false
	}

	return true
}

func subtractTime(time1, time2 time.Time) float64 {
	diff := time2.Sub(time1).Seconds()
	return diff
}
