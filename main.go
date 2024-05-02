package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/brianstrauch/spotify"
)

func getConfigFilename() string {
	var runtime, _ = os.LookupEnv("XDG_CONFIG_HOME")
	return filepath.Join(runtime, "spotify-mpris", "config.conf")
}

func getConfig() map[string]string {
	filename := getConfigFilename()
	file, err := os.Open(filename)
	if err != nil {
		log.Println("Error opening file:", err)
		return make(map[string]string, 0)
	}

	log.Printf("Reading %s", filename)

	// Create a map to store the key-value pairs
	config := make(map[string]string)

	// Create a scanner to read the file line by line
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// Split each line into key-value pairs
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			config[key] = value
		}
	}

	return config

}

func saveConfig(name string, value string) {
	config := getConfig()
	config[name] = value

	file, err := os.Create(getConfigFilename())
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	// Iterate over the map and write key-value pairs to the file
	for key, value := range config {
		_, err := fmt.Fprintf(file, "%s=%s\n", key, value)
		if err != nil {
			fmt.Println("Error writing to file:", err)
			continue
		}
	}

}

func getCredential(name string, defaultValue string, mandatory bool) string {
	credentials, exists := os.LookupEnv("SPOTIFY_MPRIS_" + name)
	if exists {
		return credentials
	}

	var config = getConfig()

	value, exists := config[name]

	if !exists {
		if !mandatory {
			log.Printf("Unable to fetch credential %s", name)
			return defaultValue
		}

		log.Fatalf("Missing credential %s", name)
	}

	return value
}

func ListenForCode(state string) (code string, err error) {
	server := &http.Server{Addr: ":" + getCredential("PORT", "10001", false)}

	http.HandleFunc("/spotify/redirect", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state || r.URL.Query().Get("error") != "" {
			err = errors.New("authorization failed")
			fmt.Fprintln(w, "Failure.")
		} else {
			code = r.URL.Query().Get("code")
			fmt.Fprintln(w, "Success!")
		}

		// Use a separate thread so browser doesn't show a "No Connection" message
		go func() {
			server.Shutdown(context.Background())
		}()
	})

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		return "", err
	}

	return
}

func login() *spotify.Token {

	var clientId = getCredential("CLIENT_ID", "", true)
	var redirectUrl = getCredential("REDIRECT_URL", "", true)

	verifier, challenge, err := spotify.CreatePKCEVerifierAndChallenge()
	if err != nil {
		panic(err)
	}

	state, err := spotify.GenerateRandomState()
	if err != nil {
		panic(err)
	}

	scopes := []string{spotify.ScopeUserReadCurrentlyPlaying, spotify.ScopeUserModifyPlaybackState}

	url := spotify.BuildPKCEAuthURI(clientId, redirectUrl, challenge, state, scopes...)
	log.Printf("URL %s", url)

	code, err := ListenForCode(state)
	if err != nil {
		log.Fatal("Unable to listen for the code callback ", err)
	}

	token, err := spotify.RequestPKCEToken(clientId, code, redirectUrl, verifier)
	if err != nil {
		panic(err)
	}

	return token
}

func main() {
	var token = getCredential("ACCESS_TOKEN", "", false)

	for {
		if token == "" {
			loginToken := login()
			saveConfig("ACCESS_TOKEN", loginToken.AccessToken)
			saveConfig("REFRESH_TOKEN", loginToken.RefreshToken)

		}

		log.Printf("Token: %s", token)
		api := spotify.NewAPI(token)

		playback, err := api.GetPlayback()
		if err != nil {
			log.Print("Playback issue: ", err)
			saveConfig("ACCESS_TOKEN", "")
			token = ""
			continue
		}

		fmt.Printf("Playing %s\n", playback.Item.Name)
		break
	}

}
