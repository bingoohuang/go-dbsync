package main

import (
	"github.com/emicklei/go-restful"
	"net/http"
	"log"
	"./myutil"
	"flag"
	"github.com/dgiagio/getpass"
)

func NewRestMi() *restful.WebService {
	service := new(restful.WebService)
	service.Path("/mi").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	service.Route(service.GET("/mi/{record}").To(MiRecord))
	service.Route(service.GET("/im/{record}").To(ImRecord))

	return service
}

var keyStr string

func MiRecord(request *restful.Request, response *restful.Response) {
	record := request.PathParameter("record")
	usr, _ := myutil.CBCEncrypt(keyStr, record)
	response.WriteEntity(usr)
}

func ImRecord(request *restful.Request, response *restful.Response) {
	record := request.PathParameter("record")
	usr, _ := myutil.CBCDecrypt(keyStr, record)
	response.WriteEntity(usr)
}

func main() {
	key := flag.String("key", "", "key to encyption or decyption")
	flag.Parse()
	keyStr := *key
	if *key == "" {
		keyStr, _ = getpass.GetPassword("Please input the key: ")
	}
	keyStr = myutil.FixStrLength(keyStr, 16)

	log.Println("key:", keyStr)
	restful.Add(NewRestMi())
	log.Fatal(http.ListenAndServe(":8080", nil))
}

