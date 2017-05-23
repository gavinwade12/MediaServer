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
	authentication *Authentication
	configuration  *Configuration
	workers        []*worker
)

func init() {
	// It's fine if we just shut down for now on error getting the config info setup.
	// In the future, we could just set up some defaults in here.
	configurationFile, err := os.Open(configurationFileName)
	if err != nil {
		log.Printf("Error reading config file: %s\nShutting down.\n", err.Error())
		panic(err)
	}

	if err = json.NewDecoder(configurationFile).Decode(&configuration); err != nil {
		log.Printf("Error decoding config file: %s\nShutting down.\n", err.Error())
		panic(err)
	}

	credentialsfile, err := os.Open(configuration.CredentialsFileName)
	if err != nil {
		log.Printf("Error reading credentials file: %s\nShutting down.\n", err.Error())
		panic(err)
	}

	if err = json.NewDecoder(credentialsfile).Decode(&authentication); err != nil {
		log.Printf("Error decoding credentials file: %s\nShutting down.\n", err.Error())
		panic(err)
	}

	for i := 0; i < configuration.MaxWorkers; i++ {
		w := newWorker()
		workers = append(workers, w)
		w.start()
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
