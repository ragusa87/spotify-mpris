package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/brianstrauch/spotify"
)

func waitAndRefreshToken(token *LoginToken, songInformation *SongInformation) {
	sleepTime := int64(5000)
	if songInformation.isPlaying {
		remaningMs := songInformation.duration - int64(songInformation.progressMs)
		sleepTime = int64(remaningMs / 100 * 10)
	}

	refreshIfNeeded(token, sleepTime)
	time.Sleep(time.Duration(sleepTime) * time.Millisecond)
}

func main() {
	songInformation := SongInformation{
		isPlaying:  false,
		title:      "",
		duration:   0,
		progressMs: 0,
	}
	loginToken, err := getToken()
	if err != nil {
		log.Printf("Get token error: %s", err)
	}

	for {
		if loginToken == nil {
			var err error
			loginToken, err = login()

			if err != nil {
				log.Printf("Login error %s", err.Error())
				loginToken = nil
				continue
			}
			saveToken(loginToken)
		}

		log.Printf("Token %s. Expire %d", loginToken.Token.AccessToken, loginToken.CreatedAt+int64(loginToken.Token.ExpiresIn))
		api := spotify.NewAPI(loginToken.Token.AccessToken)
		playback, err := api.GetPlayback()

		// If there is an error, reset AUTH
		if err != nil && !strings.Contains(err.Error(), "No active device found") {
			log.Printf("Playback issue, reseting token. Error: %s", err)
			loginToken = nil
			continue
		}

		// No active device
		if err != nil && strings.Contains(err.Error(), "No active device found") {
			log.Printf("Nothing is playing")
			songInformation.isPlaying = false
			songInformation.duration = 0
			songInformation.progressMs = 0
			waitAndRefreshToken(loginToken, &songInformation)
			continue
		}

		songInformation.isPlaying = playback.IsPlaying
		songInformation.title = playback.Item.Name
		songInformation.duration = time.Duration.Milliseconds(playback.Item.Duration.Duration)
		songInformation.progressMs = int64(playback.ProgressMs)

		if songInformation.isPlaying {
			fmt.Printf("Playing %s\n", songInformation.title)
			remaningMs := songInformation.duration - int64(songInformation.progressMs)
			fmt.Printf("Remaining %d ms\n", remaningMs)
		} else {
			fmt.Printf("Paused %s\n", songInformation.title)

		}

		// Wait a bit before fetching information from Spotify API
		waitAndRefreshToken(loginToken, &songInformation)
		if loginToken == nil {
			continue
		}

		// TODO Connect to DBUS, instanciate Mpris and pass songInformation
	}

}
