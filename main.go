package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
)

var (
	token string = "MTIxNzEyNDY3MTI1MDQ5NzYwOA.GW_mLk.USliS6dP9BLb_8jeZwZSUnajY_1uTUCOERWkeM"
	appID string = "1217124671250497608"
)

func main() {
	// Create a new Discord session
	dg, err := discordgo.New("Bot " + token)
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
		log.Fatalf("Cannot open the session: %v", err)
	}

	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
	for i, v := range commands {
		cmd, err := dg.ApplicationCommandCreate(dg.State.User.ID, "", v)
		if err != nil {
			log.Panicf("Cannot create '%v' command: %v", v.Name, err)
		}
		registeredCommands[i] = cmd
	}

	defer dg.Close()

	// Keep the bot running until interrupted
	fmt.Println("Bot is now running. Press CTRL+C to exit.")
	<-make(chan struct{})
}
