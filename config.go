package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"errors"

	"github.com/brianstrauch/spotify"
)

func getConfigFilename() string {
	var runtime, exists = os.LookupEnv("XDG_CONFIG_HOME")
	if !exists {
		runtime = ".config"
	}
	return filepath.Join(runtime, "spotify-mpris", "config.conf")
}

type Config map[string]string

func getConfig() Config {
	filename := getConfigFilename()
	file, err := os.Open(filename)
	if err != nil {
		log.Println("Error opening file:", err)
		return make(Config, 0)
	}

	// Create a map to store the key-value pairs
	config := make(Config)

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
func saveConfigCallback(callback func(Config) Config) {
	config := getConfig()
	config = callback(config)
	saveConfigs(config)
}

func saveConfig(name string, value string) {
	saveConfigCallback(func(config Config) Config {
		config[name] = value
		return config
	})

}

func saveConfigs(config Config) {
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

func saveToken(token *LoginToken) {

	if token == nil {
		saveConfig("TOKEN", "")
		return
	}

	// Serialize the Token struct to JSON
	jsonData, err := json.Marshal(token)
	if err != nil {
		log.Fatal("Error marshalling JSON:", err)
		return
	}

	saveConfig("TOKEN", string(jsonData))
}

func getToken() (*LoginToken, error) {

	tokenScruct := new(LoginToken)
	spotifyToken := new(spotify.Token)
	tokenScruct.Token = spotifyToken
	tokenScruct.CreatedAt = 0

	tokenRaw := getConfigValue("TOKEN", "", false)
	if tokenRaw == "" {
		return nil, errors.New("no token was stored")
	}

	reader := strings.NewReader(tokenRaw)
	err := json.NewDecoder(reader).Decode(&tokenScruct)
	if err != nil {
		return nil, errors.New("error decoding token")
	}

	return tokenScruct, nil
}

func getConfigValue(name string, defaultValue string, mandatory bool) string {
	environmentValue, exists := os.LookupEnv("SPOTIFY_MPRIS_" + name)
	if exists {
		return environmentValue
	}

	var config = getConfig()
	value, exists := config[name]

	if !exists {
		if !mandatory {
			return defaultValue
		}

		log.Fatalf("Missing config %s", name)
	}

	return value
}
