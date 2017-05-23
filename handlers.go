package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	maxMemory               = 32 << 20
	formFileName            = "file"
	fileFromFormErrorFormat = "Error encountered getting file from form: %s\n"
	openDestFileErrorFormat = "Error opening destination file: %s\n"
	badMethodErrorFormat    = "Request with bad method. Method: %s, Path: %s\n"
	authCookieBadValueError = "Auth cookie value does not contain correct format. Value: %s"
	authCookieBadCredError  = "Auth cookie contains "
	authCookieName          = "nfsa-auth"
	authCookieDuration      = time.Minute * 20
)

var templates *template.Template

func init() {
	// TODO: Obviously these could be a little prettier, but that's pretty low priority,
	// not Go, and 'that gay css styling shit'
	templates = template.Must(template.ParseFiles("login.html", "upload.html"))
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
}

func uploadMedia(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		err := templates.ExecuteTemplate(w, "upload.html", nil)
		if err != nil {
			log.Printf("Failed on template execution: %s, template: upload.html\n", err.Error())
			http.Error(w, "Couldn't render the requested page.", http.StatusInternalServerError)
		}
		return
	}

	if r.Method != http.MethodPost {
		log.Printf(badMethodErrorFormat, r.Method, r.URL.Path)
		http.NotFound(w, r)
		return
	}

	r.ParseMultipartForm(maxMemory)
	uploadFile, header, err := r.FormFile(formFileName)
	if err != nil {
		log.Printf(fileFromFormErrorFormat, err.Error())
		http.Error(w, fmt.Sprintf(fileFromFormErrorFormat, err.Error()), http.StatusInternalServerError)
		return
	}
	log.Printf("Uploading file: %s\n", header.Filename)
	defer uploadFile.Close()

	// TODO: Change the directory structure of uploaded files. It should be something like:
	// `mediaDirectoryPath`/<year>/<month>/<day>/fileExtension/fileName
	// ex. `mediaDirectoryPath`/2017/05/05/NEF/newPhoto.NEF
	destFilePath := configuration.MediaDirectoryPath + header.Filename
	if _, err := os.Stat(destFilePath); err == nil {
		log.Printf("Renaming due to existing file: %s", header.Filename)
		for count := 1; !os.IsNotExist(err); count++ {
			filename := header.Filename[:len(header.Filename)-4]
			filename += "(" + strconv.Itoa(count) + ")"
			filename += header.Filename[len(header.Filename)-4:]
			destFilePath = configuration.MediaDirectoryPath + filename
			_, err = os.Stat(destFilePath)
		}
	}

	destFile, err := os.OpenFile(destFilePath, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		log.Printf(openDestFileErrorFormat, err.Error())
		http.Error(w, fmt.Sprintf(openDestFileErrorFormat, err.Error()), http.StatusInternalServerError)
		return
	}
	defer destFile.Close()

	io.Copy(destFile, uploadFile)
	log.Printf("File uploaded: %s", destFilePath)

	if strings.HasSuffix(strings.ToLower(destFilePath), ".nef") {
		log.Printf("Adding file to conversion queue: %s\n", destFilePath)
		workQueue <- destFilePath
	}
}

func login(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		err := templates.ExecuteTemplate(w, "login.html", nil)
		if err != nil {
			log.Printf("Failed on template execution: %s, template: login.html\n", err.Error())
			http.Error(w, "Couldn't render the requested page.", http.StatusInternalServerError)
		}
		return
	}

	if r.Method != http.MethodPost {
		log.Printf(badMethodErrorFormat, r.Method, r.URL.Path)
		http.NotFound(w, r)
		return
	}

	if err := r.ParseForm(); err != nil {
		log.Printf("Error parsing form: %s\n", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	u := r.Form.Get("username")
	p := r.Form.Get("password")
	if u != authentication.Username || p != authentication.Password {
		log.Printf("Bad login attempt. Username: %s, Password: %s\n", u, p)
		http.Error(w, "Username and password don't match", http.StatusUnauthorized)
		return
	}

	cookie := &http.Cookie{
		Name:    authCookieName,
		Expires: time.Now().Add(authCookieDuration).UTC(),
		Value:   authentication.CookieValue,
	}
	http.SetCookie(w, cookie)
	log.Println("Successful Login.")
	w.Write([]byte(`<a href="/upload">Upload</a>`))
}
