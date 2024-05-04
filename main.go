package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"github.com/zmb3/spotify/v2"
	"log"
	"os"
	"sync"
)

var (
	TOKEN    string
	speaking bool
	mu       sync.Mutex
	spClient *spotify.Client
)

const botID = "1217124671250497608"

func main() {
	if err := godotenv.Load("./.env"); err != nil {
		log.Fatal("Error loading .env file", err)
	}
	spClient = initializeSpotifyClient()

	file, err := os.OpenFile("logs.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	log.SetOutput(file)

	TOKEN = os.Getenv("TOKEN")

	// Create a new Discord session
	dg, err := discordgo.New("Bot " + TOKEN)
	if err != nil {
		fmt.Println("Error creating Discord session: ", err)
		return
	}

	dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})

	dg.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
	})
	err = dg.Open()
	if err != nil {
		log.Fatalf("Cannot++ open the session: %v", err)
	}

	dg.AddHandler(onVoiceStateUpdate)

	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
	for i, v := range commands {
		cmd, err := dg.ApplicationCommandCreate(dg.State.User.ID, "", v)
		if err != nil {
			log.Panicf("Cannot create '%v' command: %v", v.Name, err)
		}
		registeredCommands[i] = cmd
	}

	defer func(dg *discordgo.Session) {
		err := dg.Close()
		if err != nil {
			log.Println(err)
			return
		}
	}(dg)

	// Keep the bot running until interrupted
	log.Println("Bot is now running. Press CTRL+C to exit.")
	<-make(chan struct{})
}
