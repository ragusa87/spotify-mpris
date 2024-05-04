package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/brianstrauch/spotify"
)

func wait(ms int) {
	log.Printf("Waiting %d ms", ms)
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

func main() {
	songInformation := SongInformation{
		isPlaying: false,
		title:     "",
	}
	loginToken, err := getToken()
	if err != nil {
		log.Printf("Error %s", err)
	}

	for {
		if loginToken == nil {
			loginToken, error := login()
			if error != nil {
				fmt.Printf("Error: %s", error)
				loginToken = nil
				continue
			}
			saveToken(loginToken)
		}

		log.Printf("Token %s", loginToken.AccessToken)
		api := spotify.NewAPI(loginToken.AccessToken)
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
			wait(5000)
			continue
		}

		songInformation.isPlaying = playback.IsPlaying
		songInformation.title = playback.Item.Name

		if songInformation.isPlaying {
			fmt.Printf("Playing %s\n", songInformation.title)
			remainingMs := time.Duration.Milliseconds(playback.Item.Duration.Duration) - int64(playback.ProgressMs)

			fmt.Printf("Remaining %d ms\n", remainingMs)
			wait(int(remainingMs / 100.0 * 10.0))

		} else {
			fmt.Printf("Paused %s\n", songInformation.title)
			wait(5000)
		}

		// TODO Connect to DBUS, instanciate Mpris and pass songInformation
	}

}
