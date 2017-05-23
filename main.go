package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

const (
	configurationFileName = "config.json"
)

// Configuration struct for containing values from configuration file.
type Configuration struct {
	CredentialsFileName string
	MaxWorkers          int
	MediaDirectoryPath  string
}

// Authentication struct for containing values for authentication file.
type Authentication struct {
	CookieValue string
	Password    string
	Username    string
}

var (
	authentication Authentication
	configuration  Configuration
	workers        []*worker
)

func init() {
	configurationFile, error := os.Open(configurationFileName)
	if error != nil {
		log.Printf("Error reading file: %s\n", error.Error())
	} else {
		configurationDecoder := json.NewDecoder(configurationFile)
		configuration := Configuration{}
		err := configurationDecoder.Decode(&configuration)
		log.Printf("configuration: %v\n", configuration)
		if err != nil {
			log.Printf("Error decoding file: %s\n", error.Error())
		} else {
			credentialsfile, error := os.Open(configuration.CredentialsFileName)
			if error != nil {
				log.Printf("Error reading file: %s\n", error.Error())
			} else {
				credentialsDecoder := json.NewDecoder(credentialsfile)
				authentication := Authentication{}
				err := credentialsDecoder.Decode(&authentication)
				log.Printf("authentication: %v\n", authentication)

				if err != nil {
					log.Printf("Error decoding file: %s\n", error.Error())
				} else {
					for i := 0; i < configuration.MaxWorkers; i++ {
						w := newWorker()
						workers = append(workers, w)
						w.start()
					}
				}
			}
		}
	}
}

func main() {
	http.Handle("/healthcheck", http.HandlerFunc(healthCheck))
	http.Handle("/upload", cookieCheckMiddleware(http.HandlerFunc(uploadMedia)))
	http.Handle("/login", http.HandlerFunc(login))
	http.Handle("/", cookieCheckMiddleware(http.FileServer(http.Dir(configuration.MediaDirectoryPath))))
	log.Fatal(http.ListenAndServe(":1998", nil))
}

func isValidCookie(r *http.Request) bool {
	cookie, err := r.Cookie(authCookieName)
	if err != nil {
		log.Printf("Error retrieving auth cookie: %s\n", err.Error())
		return false
	}

	if cookie.Value != authentication.CookieValue {
		log.Printf("Invalid cookie value: %s\n", cookie.Value)
		return false
	}

	return true
}

func cookieCheckMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isValidCookie(r) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		h.ServeHTTP(w, r)
	})
}
