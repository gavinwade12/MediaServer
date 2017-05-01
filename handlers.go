package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
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

	cookie, err := r.Cookie(authCookieName)
	if err != nil {
		log.Printf("Error retrieving auth cookie: %s\n", err.Error())
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if cookie.Value != cookieValue {
		log.Printf("Invalid cookie value: %s\n", cookie.Value)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
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

	destFile, err := os.OpenFile(mediaDirectoryPath+header.Filename, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		log.Printf(openDestFileErrorFormat, err.Error())
		http.Error(w, fmt.Sprintf(openDestFileErrorFormat, err.Error()), http.StatusInternalServerError)
		return
	}
	defer destFile.Close()

	io.Copy(destFile, uploadFile)
	log.Printf("File uploaded: %s", header.Filename)

	if strings.HasSuffix(strings.ToLower(header.Filename), ".nef") {
		file := mediaDirectoryPath + header.Filename
		log.Printf("Adding file to conversion queue: %s\n", file)
		workQueue <- file
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
	if u != username || p != password {
		log.Printf("Bad login attempt. Username: %s, Password: %s\n", u, p)
		http.Error(w, "Username and password don't match", http.StatusUnauthorized)
		return
	}

	cookie := &http.Cookie{
		Name:    authCookieName,
		Expires: time.Now().Add(authCookieDuration).UTC(),
		Value:   cookieValue,
	}
	http.SetCookie(w, cookie)
	log.Println("Successful Login.")
	w.Write([]byte(`<a href="/upload">Upload</a>`))
}
