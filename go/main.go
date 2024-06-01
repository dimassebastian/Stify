package main

import (
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"text/template"

	spotifyauth "Spotify/auth" // Ensure the import path is correct

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/sessions"
	"github.com/zmb3/spotify/v2"
	"golang.org/x/oauth2"
)

const redirectURI = "http://localhost:8080/callback"

var (
	auth  = spotifyauth.New(spotifyauth.WithRedirectURL(redirectURI), spotifyauth.WithScopes(spotifyauth.ScopeUserReadPrivate, spotifyauth.ScopeUserTopRead))
	store = sessions.NewCookieStore([]byte("something-very-secret"))
	state = "abc123"
	// These should be securely generated for each request
	codeVerifier  = "W-H0DQRVeeQbIWOETsQq3Q67bLvythaQWsIicV7eBceK3n6g"
	codeChallenge = "pxscqKGikXV8CaF-04XRB_odrRJaqrJM_TbTUzLxHPI"
)

func init() {
	// Register the oauth2.Token type with gob
	gob.Register(&oauth2.Token{})
}

func main() {
	r := chi.NewRouter()

	// Serve static files from the "public" directory
	fileServer := http.FileServer(http.Dir("../public"))
	r.Handle("/public/*", http.StripPrefix("/public", fileServer))

	r.Get("/callback", completeAuth)
	r.Get("/", homeHandler)
	r.Get("/home", homePageHandler)
	r.Get("/user-info", userInfoHandler)
	r.Get("/top-tracks", topTracksHandler)
	r.Get("/top-tracks2", topTracksHandler2)
	r.Get("/top-tracks3", topTracksHandler3)

	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatal(err)
	}

	url := auth.AuthURL(state,
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
	)
	fmt.Println("Please log in to Spotify by visiting the following page in your browser:", url)
}

func completeAuth(w http.ResponseWriter, r *http.Request) {
	tok, err := auth.Token(r.Context(), state, r,
		oauth2.SetAuthURLParam("code_verifier", codeVerifier))
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusForbidden)
		log.Printf("Error getting token: %v", err)
		return
	}
	if st := r.FormValue("state"); st != state {
		http.NotFound(w, r)
		log.Printf("State mismatch: %s != %s\n", st, state)
		return
	}

	// Save token to session
	session, _ := store.Get(r, "auth-session")
	session.Values["token"] = tok
	err = session.Save(r, w)
	if err != nil {
		http.Error(w, "Couldn't save session", http.StatusInternalServerError)
		log.Printf("Error saving session: %v", err)
		return
	}
	log.Printf("Token saved: %v\n", tok)

	// Redirect to home page
	http.Redirect(w, r, "/home", http.StatusSeeOther)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	url := auth.AuthURL(state,
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
	)

	tmpl, err := template.ParseFiles("../public/templates/index.html")
	if err != nil {
		http.Error(w, "Couldn't load template", http.StatusInternalServerError)
		log.Printf("Error loading template: %v", err)
		return
	}

	data := struct {
		AuthURL string
	}{
		AuthURL: url,
	}

	w.Header().Set("Content-Type", "text/html")
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Couldn't render template", http.StatusInternalServerError)
		log.Printf("Error rendering template: %v", err)
	}
}

func homePageHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "auth-session")
	tok, ok := session.Values["token"].(*oauth2.Token)
	if !ok || tok == nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Print token to check if it's valid
	log.Printf("Token from session: %v\n", tok)

	client := spotify.New(auth.Client(r.Context(), tok))
	user, err := client.CurrentUser(context.Background())
	if err != nil {
		http.Error(w, "Couldn't get user", http.StatusInternalServerError)
		log.Printf("Error getting user: %v", err)
		return
	}

	tmpl, err := template.ParseFiles("../public/templates/main.html")
	if err != nil {
		http.Error(w, "Couldn't load template", http.StatusInternalServerError)
		log.Printf("Error loading template: %v", err)
		return
	}

	data := struct {
		UserID string
	}{
		UserID: user.ID,
	}

	w.Header().Set("Content-Type", "text/html")
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Couldn't render template", http.StatusInternalServerError)
		log.Printf("Error rendering template: %v", err)
	}
}

func userInfoHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "auth-session")
	tok, ok := session.Values["token"].(*oauth2.Token)
	if !ok || tok == nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	client := spotify.New(auth.Client(r.Context(), tok))
	user, err := client.CurrentUser(context.Background())
	if err != nil {
		http.Error(w, "Couldn't get user", http.StatusInternalServerError)
		log.Printf("Error getting user: %v", err)
		return
	}

	// Write the user's display name directly to the response
	w.Header().Set("Content-Type", "text/plain")
	if _, err := fmt.Fprintf(w, user.DisplayName); err != nil {
		http.Error(w, "Couldn't write user information to response", http.StatusInternalServerError)
		log.Printf("Error writing user information to response: %v", err)
	}
}

type Track struct {
	ImageURL string `json:"img"`
	Title    string `json:"title"`
	Artist   string `json:"artist"`
}

func topTracksHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "auth-session")
	tok, ok := session.Values["token"].(*oauth2.Token)
	if !ok || tok == nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Print token to check if it's valid
	log.Printf("Token from session: %v\n", tok)

	client := spotify.New(auth.Client(r.Context(), tok))

	// Get top tracks
	options := []spotify.RequestOption{
		spotify.Limit(15),                         // Batasan jumlah (10 lagu)
		spotify.Offset(0),                         // Mulai dari indeks 0
		spotify.Timerange(spotify.ShortTermRange), // Rentang waktu: Pendek
		// spotify.TimeRange(spotify.MediumTermRange),  // Rentang waktu: Menengah
		// spotify.TimeRange(spotify.LongTermRange),    // Rentang waktu: Panjang
	}

	tracks, err := client.CurrentUsersTopTracks(context.Background(), options...)
	if err != nil {
		log.Fatalf("failed to get topTracks: %v", err)
	}

	var trackList []Track
	for _, item := range tracks.Tracks {
		track := Track{
			ImageURL: item.Album.Images[0].URL,
			Title:    item.Name,
			Artist:   item.Artists[0].Name,
		}
		trackList = append(trackList, track)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(trackList)
}

func topTracksHandler2(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "auth-session")
	tok, ok := session.Values["token"].(*oauth2.Token)
	if !ok || tok == nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Print token to check if it's valid
	log.Printf("Token from session: %v\n", tok)

	client := spotify.New(auth.Client(r.Context(), tok))

	// Get top tracks
	options := []spotify.RequestOption{
		spotify.Limit(15),                          // Batasan jumlah (10 lagu)
		spotify.Offset(0),                          // Mulai dari indeks 0
		spotify.Timerange(spotify.MediumTermRange), // Rentang waktu: Pendek
		// spotify.TimeRange(spotify.MediumTermRange),  // Rentang waktu: Menengah
		// spotify.TimeRange(spotify.LongTermRange),    // Rentang waktu: Panjang
	}

	tracks, err := client.CurrentUsersTopTracks(context.Background(), options...)
	if err != nil {
		log.Fatalf("failed to get topTracks: %v", err)
	}

	var trackList []Track
	for _, item := range tracks.Tracks {
		track := Track{
			ImageURL: item.Album.Images[0].URL,
			Title:    item.Name,
			Artist:   item.Artists[0].Name,
		}
		trackList = append(trackList, track)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(trackList)
}

func topTracksHandler3(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "auth-session")
	tok, ok := session.Values["token"].(*oauth2.Token)
	if !ok || tok == nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Print token to check if it's valid
	log.Printf("Token from session: %v\n", tok)

	client := spotify.New(auth.Client(r.Context(), tok))

	// Get top tracks
	options := []spotify.RequestOption{
		spotify.Limit(15),                        // Batasan jumlah (10 lagu)
		spotify.Offset(0),                        // Mulai dari indeks 0
		spotify.Timerange(spotify.LongTermRange), // Rentang waktu: Pendek
		// spotify.TimeRange(spotify.MediumTermRange),  // Rentang waktu: Menengah
		// spotify.TimeRange(spotify.LongTermRange),    // Rentang waktu: Panjang
	}

	tracks, err := client.CurrentUsersTopTracks(context.Background(), options...)
	if err != nil {
		log.Fatalf("failed to get topTracks: %v", err)
	}

	var trackList []Track
	for _, item := range tracks.Tracks {
		track := Track{
			ImageURL: item.Album.Images[0].URL,
			Title:    item.Name,
			Artist:   item.Artists[0].Name,
		}
		trackList = append(trackList, track)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(trackList)
}
