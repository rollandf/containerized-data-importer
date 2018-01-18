package controller

import (
	//"log"
	//"net/http"
	_ "context"
	b64 "encoding/base64"
	"flag"
	_ "fmt"
	"github.com/golang/glog"
	"os"
)

var tmp_endpoint, _ = b64.StdEncoding.DecodeString(os.Getenv("BUCKET_ENDPOINT"))
var tmp_accessKeyID, _ = b64.StdEncoding.DecodeString(os.Getenv("BUCKET_ID"))
var tmp_secretAccessKey, _ = b64.StdEncoding.DecodeString(os.Getenv("BUCKET_PWORD"))
var endpoint = string(tmp_endpoint[:])
var accessKeyID = string(tmp_accessKeyID[:])
var secretAccessKey = string(tmp_secretAccessKey[:])

var options struct {
	Port int
}

func init() {
	flag.IntVar(&options.Port, "port", 8005, "use '--port' option to specify the port for broker to listen on")
	flag.Parse()
}

func main() {

	defer glog.Flush()

	//----------- CONTROLLER --------------
	//images := GetImagesFromObjStore()
	//if HaveImagesChanged(images) {
	//
	//
	//	pvc := CreatePVC()	//Create Golden PVC that will hold VM image
	//	secret := CreateSecret()  //Get credentials for obj storage to give to Pod s3.awsamazon.com/bucketname/image.img plus credentials
	//	copyPod := CreateCopyPod(pvc,secret) //Pass in secret and create Pod to copy into Golden PVC
	//
	//	if copyPod == 0 {
	//
	//	}
	//}

	//---------- COPIER --------------------------
	//1)Setup the environment
	//2) create io buffer and copy the img into it
	//3) Save that buffer into mnt

	glog.Infoln("List all pods\n")
	ListPods()

	//router := NewRouter()
	//
	//log.Fatal(http.ListenAndServe(":8888", router))
}