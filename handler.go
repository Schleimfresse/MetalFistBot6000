package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
	"strconv"
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
func handleQueuePage(s *discordgo.Session, i *discordgo.InteractionCreate) {
	page := 0
	start := (page - 1) * 10
	end := page * 10
	if end > len(queue) {
		end = len(queue)
	}

	user, err := s.User("@me")
	avatarUrl := user.AvatarURL("")

	embed := &discordgo.MessageEmbed{
		Title:     "Queue",
		Timestamp: timestamp(i.Interaction),
		Color:     botThemeColor,
		Author:    &discordgo.MessageEmbedAuthor{Name: "MetalFistBot 6000", IconURL: avatarUrl},
		Footer:    &discordgo.MessageEmbedFooter{Text: fmt.Sprint("Requested by ", i.Member.User.Username)},
	}

	for i := start; i < end; i++ {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   strconv.Itoa(i + 1),
			Value:  fmt.Sprint(queue[i].title, " - ", formatDuration(queue[0].duration)),
			Inline: false,
		})
	}

	log.Println("INDEXES:", start, end, page)

	minSelected := 1
	pages := discordgo.SelectMenu{MinValues: &minSelected, MaxValues: 1, CustomID: "queuePage"}

	for i := 0; i < totalPages(queue); i++ {
		pages.Options = append(pages.Options, discordgo.SelectMenuOption{
			Label: fmt.Sprint("Page ", i+1),
			Value: strconv.Itoa(i),
			Emoji: discordgo.ComponentEmoji{Name: ":queue_play_next_24dp_FILL0_wght4", ID: "1237399904846876692"},
		})
	}

	/*s.WebhookMessageEdit(data.AppID, data.Token, "@original", &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
		Components: &[]discordgo.MessageComponent{
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{pages}},
		},
	})*/

	if err != nil {
		log.Println("Error updating queue display:", err)
		return
	}
}
