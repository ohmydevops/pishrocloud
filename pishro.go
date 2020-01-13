package pishrocloud

import (
	"io"
	"log"
	"net/http"
	"os"
)

// Storage ...
type Storage struct {
	APIKey           string
	AuthURL          string
	SwiftURL         string
	DefaultContainer string // todo: add default container
}

// MakeRequest ...
func MakeRequest(Method string, URL string, Token string, Headers map[string]string, Object io.Reader) http.Response {
	client := http.Client{}
	request, err := http.NewRequest(Method, URL, Object)

	if err != nil {
		log.Fatalln(err)
	}

	// Default Headers
	request.Header.Set("X-Auth-Token", Token)
	request.Header.Set("Content-type", "application/json")
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
func (p *Storage) IsObjectExist(fileName string, container string) bool {
	var response = MakeRequest("HEAD", p.SwiftURL+container+"/"+fileName, p.APIKey, nil, nil)
	statusCode := response.StatusCode
	if statusCode != 200 {
		return false
	}
	// fmt.Printf("%v", response.Header) : for see metadata
	return true
}
