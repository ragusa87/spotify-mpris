package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/brianstrauch/spotify"
)

type LoginToken struct {
	Token     *spotify.Token
	CreatedAt int64
}

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
	fmt.Printf("Code received: %s\n", code)
	return code, nil
}

func login() (*LoginToken, error) {

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
	log.Printf("Requesting PKCE token..")
	token, err := spotify.RequestPKCEToken(clientId, code, redirectUrl, verifier)
	if err != nil || token == nil {
		log.Printf("Error while requesting PKCE Token")
		return nil, err
	}
	// Refresh the token directly
	log.Printf("Refreshing PKCE token..")
	token, err = spotify.RefreshPKCEToken(token.RefreshToken, clientId)
	if err != nil || token == nil {
		log.Printf("Error while refreshing PKCE Token")
		return nil, err
	}

	loginToken := new(LoginToken)
	loginToken.Token = token
	loginToken.CreatedAt = time.Now().Unix()

	log.Printf("New token %s", loginToken.Token.AccessToken)

	return loginToken, nil
}

func refreshToken(token *LoginToken) error {
	// Refresh the token directly
	if token == nil || token.Token == nil || token.Token.RefreshToken == "" {
		log.Printf("Empty refresh token provided")
		token = nil
	}
	var clientId = getConfigValue("CLIENT_ID", "", true)

	log.Printf("Refreshing token")
	spotifyToken, err := spotify.RefreshPKCEToken(token.Token.RefreshToken, clientId)
	if err != nil {
		return err
	}

	if spotifyToken == nil {
		return errors.New("null refresh token received")
	}

	token.Token = spotifyToken
	token.CreatedAt = time.Now().Unix()

	return nil
}

func refreshIfNeeded(token *LoginToken, sleepTime int64) bool {
	const marginSec = 5

	remaining := token.CreatedAt + int64(token.Token.ExpiresIn) - time.Now().Unix() - marginSec
	log.Printf("Token remaning time: %d sec, sleep for %d sec", remaining, sleepTime/1000)

	if remaining-int64(sleepTime/1000) < 0 {
		error := refreshToken(token)
		if error != nil {
			log.Printf("Error while refreshing token %s", error.Error())
			token = nil
		}
		return true
	}

	return false
}
