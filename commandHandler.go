package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/kkdai/youtube/v2"
	"log"
	"math/rand"
	"regexp"
	"strconv"
	"time"
)

var (
	queue      []ytTrack
	connection *discordgo.VoiceConnection
	ytRegex    = regexp.MustCompile("^(https?://)?(www\\.)?(youtube\\.com|music\\.youtube\\.com)/.*$")
)

type ytTrack struct {
	title       string
	streamUrl   string
	duration    time.Duration
	author      string
	publishDate time.Time
}

func playHandler(s *discordgo.Session, i *discordgo.InteractionCreate, playNext bool) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		log.Println(err)
		return
	}

	snowflakeTimestamp, err := discordgo.SnowflakeTimestamp(i.Interaction.ID)
	if err != nil {
		log.Println(err)
		return
	}
	user, err := s.User("@me")
	avatarUrl := user.AvatarURL("")
	formattedTime := snowflakeTimestamp.Format("2006-01-02T15:04:05Z")
	embed := &discordgo.MessageEmbed{
		Timestamp: formattedTime,
		Author:    &discordgo.MessageEmbedAuthor{Name: "MetalFistBot 6000", IconURL: avatarUrl},
		Footer:    &discordgo.MessageEmbedFooter{Text: fmt.Sprint("Requested by ", i.Member.User.Username)},
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

			if !ytRegex.MatchString(videoUrl) {
				s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
					Content: "Please use a valid link",
				})
				return
			}

			client := youtube.Client{}

			video, err := client.GetVideo(videoUrl)
			if err != nil {
				log.Println(err)
			}

			format := findBestAudioFormat(video.Formats)
			if format == nil {
				log.Fatal("No audio format found")
			}

			url, err := client.GetStreamURL(video, format)
			track := ytTrack{title: video.Title, streamUrl: url, duration: video.Duration, author: video.Author, publishDate: video.PublishDate}

			if playNext {
				insertIndex := 1
				for i := len(queue) - 1; i > insertIndex; i-- {
					queue[i] = queue[i-1]
				}
				queue[insertIndex] = track
			} else {
				queue = append(queue, track)
			}

			if !getSpeakingState() {
				embed.Description = fmt.Sprint("Playing **", track.title, "** requested by ", i.Member.User.Username)
				_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
					Embeds: []*discordgo.MessageEmbed{embed},
				})
				if err != nil {
					log.Println(err)
					return
				}
				playTrack(s, i)
			} else {
				if playNext {
					embed.Description = fmt.Sprint("**", track.title, "** is playing next, requested ", i.Member.User.Username)
				} else {
					embed.Description = fmt.Sprint("**", track.title, "** was added to the queue, by ", i.Member.User.Username)
				}
				_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
					Embeds: []*discordgo.MessageEmbed{embed},
				})
				if err != nil {
					log.Println(err)
					return
				}
			}
			return
		}
	}

	s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: "You need to be in a voice channel to use this command!",
	})
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
	"pause": func(s *discordgo.Session, i *discordgo.InteractionCreate) {},
	"shuffle": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		currentlyPlaying := queue[0]
		remainingQueue := queue[1:]

		for i := range remainingQueue {
			j := rand.Intn(len(remainingQueue) - 1)
			remainingQueue[i], remainingQueue[j] = remainingQueue[j], remainingQueue[i]
		}

		queue = append([]ytTrack{currentlyPlaying}, remainingQueue...)

		snowflakeTimestamp, err := discordgo.SnowflakeTimestamp(i.Interaction.ID)
		if err != nil {
			log.Println(err)
			return
		}
		user, err := s.User("@me")
		avatarUrl := user.AvatarURL("")
		formattedTime := snowflakeTimestamp.Format("2006-01-02T15:04:05Z")
		embed := &discordgo.MessageEmbed{
			Timestamp:   formattedTime,
			Author:      &discordgo.MessageEmbedAuthor{Name: "MetalFistBot 6000", IconURL: avatarUrl},
			Title:       "Shuffle",
			Description: "Shuffled the queue",
			Footer:      &discordgo.MessageEmbedFooter{Text: fmt.Sprint("Requested by ", i.Member.User.Username)},
		}

		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
		})
		if err != nil {
			log.Println(err)
		}
	},
	"queue": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		})
		if err != nil {
			log.Println(err)
			return
		}

		snowflakeTimestamp, err := discordgo.SnowflakeTimestamp(i.Interaction.ID)
		if err != nil {
			log.Println(err)
			return
		}
		user, err := s.User("@me")
		avatarUrl := user.AvatarURL("")
		formattedTime := snowflakeTimestamp.Format("2006-01-02T15:04:05Z")

		embed := &discordgo.MessageEmbed{
			Title:     "Queue",
			Timestamp: formattedTime,
			Author:    &discordgo.MessageEmbedAuthor{Name: "MetalFistBot 6000", IconURL: avatarUrl},
			Footer:    &discordgo.MessageEmbedFooter{Text: fmt.Sprint("Requested by ", i.Member.User.Username)},
		}
		if len(queue) <= 0 {
			embed.Description = "Queue is empty."
			s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Embeds: []*discordgo.MessageEmbed{embed},
			})
		} else {
			page := 1
			start := (page - 1) * 10
			end := page * 10
			if end > len(queue) {
				end = len(queue)
			}

			for i := start; i < end; i++ {
				embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
					Name:   strconv.Itoa(i + 1),
					Value:  queue[i].title,
					Inline: false,
				})
			}

			_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Flags:  discordgo.MessageFlagsEphemeral,
				Embeds: []*discordgo.MessageEmbed{embed},
				Components: []discordgo.MessageComponent{
					&discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							&discordgo.Button{
								Emoji: discordgo.ComponentEmoji{Name: "⬅️"},
								Style: discordgo.PrimaryButton,
								//Label:    "Previous page",
								CustomID: "prevBtn",
							},
							&discordgo.Button{
								Emoji: discordgo.ComponentEmoji{Name: "➡️"},
								Style: discordgo.PrimaryButton,
								//Label:    "Next page",
								CustomID: "nextBtn",
							},
						},
					},
				},
			})
			if err != nil {
				log.Println(err)
				return
			}

			/*switch i.MessageComponentData().CustomID {
			case "prevBtn":
				page--
				t := "d"
				_, err := s.FollowupMessageEdit(i.Interaction, i.Message.ID, &discordgo.WebhookEdit{
					Content: &t,
					Embeds:  []discordgo.MessageEmbed{embed},
				})
				if err != nil {
					log.Printf("Error responding to button click: %s", err)
				}
			case "nextBtn":
				page++
				_, err := s.FollowupMessageEdit(i.Interaction, i.Message.ID, &discordgo.WebhookEdit{
					[]*discordgo.MessageEmbed{embed},
				})
				if err != nil {
					log.Printf("Error responding to button click: %s", err)
				}
			}*/
		}
	},
	"nowplaying": func(s *discordgo.Session, i *discordgo.InteractionCreate) {

	},
	"leave": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if connection != nil && botInVoiceChannel(s, i) {
			err := connection.Disconnect()
			if err != nil {
				log.Println("could not disconnect:", err)
			}
			connection = nil
		}
	},
	"skip": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if len(queue) == 0 {
			s.ChannelMessageSend(i.ChannelID, "There are no tracks currently playing.")
			return
		}

		// Remove the current track from the queue
		queue = queue[1:]

		// Check if there are more tracks in the queue
		if len(queue) > 0 {
			err := stopPlayback(connection)
			if err != nil {
				log.Println("skip cmd:", err)
				return
			}
			playTrack(s, i)

			_, err = s.ChannelMessageSend(i.ChannelID, "Skipping current track and playing next one.")
			if err != nil {
				log.Println("Error sending message:", err)
			}
		} else {
			// No more tracks in the queue, send a message indicating that
			s.ChannelMessageSend(i.ChannelID, "There are no more tracks in the queue.")
		}

	},
	"playlists": func(s *discordgo.Session, i *discordgo.InteractionCreate) {},
}
