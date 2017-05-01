package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
)

const (
	mediaDirectoryPath  = "/home/gavin/pictures/nicole/"
	maxWorkers          = 5
	credentialsFileName = "cred.json"
)

var (
	workers     []*worker
	username    string
	password    string
	cookieValue string
)

func init() {
	for i := 0; i < maxWorkers; i++ {
		w := newWorker()
		workers = append(workers, w)
		w.start()
	}

	data, err := ioutil.ReadFile(credentialsFileName)
	if err == nil {
		temp := make(map[string]string)
		if err = json.Unmarshal(data, &temp); err != nil {
			log.Printf("Error processing credentials data: %s\n", err.Error())
		}

		username = temp["username"]
		password = temp["password"]
		cookieValue = temp["cookie"]
	} else {
		log.Printf("Error reading file: %s\n", err.Error())
	}
}

func main() {
	http.Handle("/healthcheck", http.HandlerFunc(healthCheck))
	http.Handle("/upload", http.HandlerFunc(uploadMedia))
	http.Handle("/login", http.HandlerFunc(login))
	http.Handle("/", http.FileServer(http.Dir(mediaDirectoryPath)))
	log.Fatal(http.ListenAndServe(":1998", nil))
}
