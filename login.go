package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/brianstrauch/spotify"
)

func ListenForCode(state string) (code string, err error) {
	listeningAddress := getConfigValue("SERVER_ADDR", "localhost:10001", false)
	code = ""
	mux := http.NewServeMux()
	server := &http.Server{Addr: listeningAddress}
	server.Handler = mux

	mux.HandleFunc("/spotify/redirect", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state || r.URL.Query().Get("error") != "" {
			err = errors.New("authorization failed")
			fmt.Fprintln(w, "Failure.")
		} else {
			code = r.URL.Query().Get("code")
			fmt.Fprintln(w, "Success! Your authorization code has been saved, you can close this tab")
		}

		// Use a separate to shutdown, so the browser has time to fetch the answer and doesn't show a "Unable to connect" message
		go func() {
			server.Shutdown(context.Background())
		}()
	})

	fmt.Printf("Listening at %s\n", listeningAddress)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		return "", err
	}
	fmt.Println("Code received")
	return code, nil
}

func login() (*spotify.Token, error) {

	var clientId = getConfigValue("CLIENT_ID", "", true)
	var redirectUrl = getConfigValue("REDIRECT_URL", "", true)

	verifier, challenge, err := spotify.CreatePKCEVerifierAndChallenge()
	if err != nil {
		return nil, err
	}

	state, err := spotify.GenerateRandomState()
	if err != nil {
		return nil, err
	}

	scopes := []string{spotify.ScopeUserReadCurrentlyPlaying, spotify.ScopeUserModifyPlaybackState, spotify.ScopeUserReadPlaybackState}

	url := spotify.BuildPKCEAuthURI(clientId, redirectUrl, challenge, state, scopes...)
	log.Printf("URL %s", url)

	code, err := ListenForCode(state)
	if err != nil {
		return nil, err
	}

	// Exchanges the code for an access token
	token, err := spotify.RequestPKCEToken(clientId, code, redirectUrl, verifier)
	if err != nil {
		return nil, err
	}
	// Refresh the token directly
	token, err = spotify.RefreshPKCEToken(token.RefreshToken, clientId)
	if err != nil {
		return nil, err
	}

	return token, nil

}
