package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
    // Implementa o fluxo de autorização.
    tokFile := "token.json"
    tok, err := tokenFromFile(tokFile)
    if err != nil {
        tok = getTokenFromWeb(config) // Aqui que ele processa o "code"
        saveToken(tokFile, tok)
    }
    return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("redirect_uri", "http://localhost:8080"))
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web %v", err)
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

func requestGoogle(client *http.Client) {
    // Converte o transporte para obter a fonte do token
    transport, ok := client.Transport.(*oauth2.Transport)
    if !ok {
        log.Fatalf("Unable to cast client.Transport to *oauth2.Transport")
    }

    token, err := transport.Source.Token()
    if err != nil {
        log.Fatalf("Unable to retrieve token: %v", err)
    }

    // Faz a requisição à API Google Drive usando o token
    req, err := http.NewRequest("GET", "https://www.googleapis.com/drive/v3/about?fields=user", nil)
    if err != nil {
        log.Fatalf("Unable to create request: %v", err)
    }
    req.Header.Set("Authorization", "Bearer "+token.AccessToken)

    resp, err := client.Do(req)
    if err != nil {
        log.Fatalf("Unable to perform request: %v", err)
    }
    defer resp.Body.Close()

    // Processa a resposta
    var result map[string]interface{}
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        log.Fatalf("Unable to parse response: %v", err)
    }
    fmt.Printf("Response: %v\n", result)
}

func startServer() {
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        code := r.URL.Query().Get("code")
        if code == "" {
            http.Error(w, "Code not found", http.StatusBadRequest)
            return
        }
        fmt.Fprintf(w, "Authorization successful! You can close this tab.")

        // Processar o código recebido
        fmt.Println("Authorization code:", code)
    })

    log.Println("Listening on http://localhost:8080/")
    log.Fatal(http.ListenAndServe(":8080", nil))
}


func main() {
	go startServer()
	ctx := context.Background()
	b, err := os.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, drive.DriveMetadataReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)
	requestGoogle(client)

	srv, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Drive client: %v", err)
	}

	r, err := srv.Files.List().PageSize(10).
		Fields("nextPageToken, files(id, name)").Do()
	if err != nil {
		log.Fatalf("Unable to retrieve files: %v", err)
	}
	fmt.Println("Files:")
	if len(r.Files) == 0 {
		fmt.Println("No files found.")
	} else {
		for _, i := range r.Files {
			fmt.Printf("%s (%s)\n", i.Name, i.Id)
		}
	}
}