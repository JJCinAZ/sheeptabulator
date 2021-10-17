package google

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/people/v1"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
)

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

var (
	gapiSvcInit sync.Once
	gapiSvc     *people.Service
)

func GetName(email string) (string, error) {
	var err error
	gapiSvcInit.Do(func() {
		var (
			b      []byte
			config *oauth2.Config
		)
		if b, err = ioutil.ReadFile("credentials.json"); err != nil {
			log.Fatalf("unable to read credentials.json: %s", err.Error())
		}
		config, err = google.ConfigFromJSON(b, people.ContactsReadonlyScope)
		if err != nil {
			log.Fatalf("unable to People client: %s", err.Error())
		}
		client := getClient(config)
		gapiSvc, err = people.NewService(context.Background(), option.WithHTTPClient(client))
		if err != nil {
			log.Fatalf("unable to create GAPI service: %s", err.Error())
		}
	})
	r, err := gapiSvc.People.Connections.List("people/me").
		PersonFields("names,emailAddresses").Do()
	if err != nil {
		log.Fatalf("Unable to retrieve people. %v", err)
	}
	for _, c := range r.Connections {
		var name string
		if len(c.Names) > 0 {
			name = c.Names[0].DisplayName
		}
		fmt.Printf("%s\t%s\n", c.EmailAddresses[0].Value, name)
	}
	return "me", nil
}
