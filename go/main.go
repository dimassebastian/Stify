package main

import (
	spotifyauth "Spotify/auth"
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"text/template"

	"github.com/go-chi/chi/v5"
	"github.com/zmb3/spotify/v2"
	"golang.org/x/oauth2"
)

const redirectURI = "http://localhost:8080/callback"

func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length) // Declares bytes as a slice of bytes with desired length
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

var (
	auth  = spotifyauth.New(spotifyauth.WithRedirectURL(redirectURI), spotifyauth.WithScopes(spotifyauth.ScopeUserReadPrivate))
	ch    = make(chan *spotify.Client)
	state = "abc123"
	// These should be randomly generated for each request
	//  More information on generating these can be found here,
	// https://www.oauth.com/playground/authorization-code-with-pkce.html
	codeVerifier  = "W-H0DQRVeeQbIWOETsQq3Q67bLvythaQWsIicV7eBceK3n6g"
	codeChallenge = "pxscqKGikXV8CaF-04XRB_odrRJaqrJM_TbTUzLxHPI"
)

func main() {
	r := chi.NewRouter()

	// Serve static files from the "public" directory
	fileServer := http.FileServer(http.Dir("../public"))
	r.Handle("/public/*", http.StripPrefix("/public", fileServer))

	r.Get("/callback", completeAuth)
	r.Get("/", homeHandler)

	go func() {
		log.Println("Starting server on :8080")
		if err := http.ListenAndServe(":8080", r); err != nil {
			log.Fatal(err)
		}
	}()

	url := auth.AuthURL(state,
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
	)
	fmt.Println("Please log in to Spotify by visiting the following page in your browser:", url)

	// wait for auth to complete
	client := <-ch

	// use the client to make calls that require authorization
	user, err := client.CurrentUser(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("You are logged in as:", user.ID)
}

func completeAuth(w http.ResponseWriter, r *http.Request) {
	tok, err := auth.Token(r.Context(), state, r,
		oauth2.SetAuthURLParam("code_verifier", codeVerifier))
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusForbidden)
		log.Fatal(err)
	}
	if st := r.FormValue("state"); st != state {
		http.NotFound(w, r)
		log.Fatalf("State mismatch: %s != %s\n", st, state)
	}
	// use the token to get an authenticated client
	client := spotify.New(auth.Client(r.Context(), tok))

	ch <- client

	tmpl, err := template.ParseFiles("./templates/main.html")
	if err != nil {
		http.Error(w, "Couldn't load template", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, nil); err != nil {
		http.Error(w, "Couldn't render template kucing", http.StatusInternalServerError)
	}
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	url := auth.AuthURL(state,
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
	)

	tmpl, err := template.ParseFiles("../public/index.html")
	if err != nil {
		http.Error(w, "Couldn't load template", http.StatusInternalServerError)
		return
	}

	data := struct {
		AuthURL string
	}{
		AuthURL: url,
	}

	w.Header().Set("Content-Type", "text/html")
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Couldn't render template kucing", http.StatusInternalServerError)
	}
}
