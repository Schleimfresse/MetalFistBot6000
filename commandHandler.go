package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
	"time"
)

var (
	queue      []interface{}
	connection *discordgo.VoiceConnection
)

func playHandler(s *discordgo.Session, i *discordgo.InteractionCreate, playNext bool) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}
	guild, _ := s.State.Guild(i.GuildID)
	for _, vs := range guild.VoiceStates {
		if vs.UserID == i.Member.User.ID {
			var err error
			connection, err = s.ChannelVoiceJoin(i.GuildID, vs.ChannelID, false, false)
			if err != nil {
				log.Println("Error joining vc:", err)
				return
			}

			videoUrl := i.ApplicationCommandData().Options[0].StringValue()

			err = playAudio(connection, videoUrl)
			if err == nil {
				_, err := s.ChannelMessageSend(i.ChannelID, "playing audio file!")
				if err != nil {
					log.Println(err)
				}
				return
			} else {
				log.Println(err)
			}
		}
	}

	s.ChannelMessageSend(i.ChannelID, "You need to be in a voice channel to use this command!")
}

var commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
	"play": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		playHandler(s, i, false)
	},
	"play-next": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		playHandler(s, i, true)
	},
	"ping": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		startTime := time.Now()

		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Pinging...",
			},
		})
		if err != nil {
			log.Println("Error at ping command:", err)
		}

		latency := time.Since(startTime).Milliseconds()

		res := "Ping: " + fmt.Sprint(latency) + "ms"

		_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &res})

	},
	"pause":      func(s *discordgo.Session, i *discordgo.InteractionCreate) {},
	"shuffle":    func(s *discordgo.Session, i *discordgo.InteractionCreate) {},
	"queue":      func(s *discordgo.Session, i *discordgo.InteractionCreate) {},
	"nowplaying": func(s *discordgo.Session, i *discordgo.InteractionCreate) {},
	"leave": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if connection != nil && botInVoiceChannel(s, i) {
			err := connection.Disconnect()
			if err != nil {
				log.Println("could not disconnect:", err)
			}
			connection = nil
		}
	},
	"skip":      func(s *discordgo.Session, i *discordgo.InteractionCreate) {},
	"playlists": func(s *discordgo.Session, i *discordgo.InteractionCreate) {},
}
