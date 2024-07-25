package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
)

func onVoiceStateUpdate(s *discordgo.Session, vs *discordgo.VoiceStateUpdate) {
	if autoKickState && vs.ChannelID != "" && vs.UserID != "" {
		if vs.ChannelID != "" {
			if vs.UserID == userToKick {
				err := s.GuildMemberMove(vs.GuildID, vs.UserID, nil)
				if err != nil {
					log.Println("Error kicking user:", err)
				} else {
					log.Println("Kicked user", vs.Member.User.Username, "from the voice channel")
				}
			}
		}
	}
}

func interactionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	fmt.Println("interaction create", i.Type)
	if i.Type == discordgo.InteractionMessageComponent {
		switch i.MessageComponentData().CustomID {
		case "playlistSelect":
			playHandler(s, i, false)
			_ = i.MessageComponentData().Resolved

			log.Printf("Selected playlist: %s\n", i.MessageComponentData().Values[0])
			/*case "queuePage":
			value := i.MessageComponentData().Values[0]
			var data InteractionData
			err := json.Unmarshal([]byte(value), &data)
			if err != nil {
				fmt.Println("Error:", err)
				return
			}

			handleQueuePage(s, i, data)*/
		}

	}
}
