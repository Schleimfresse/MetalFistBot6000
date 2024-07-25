package main

import (
	"MetalFistBot6000/pkg/dca"
	"database/sql"
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/kkdai/youtube/v2"
	"log"
	"math"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"time"
)

var (
	queue                  []ytTrack
	connection             *discordgo.VoiceConnection
	spTrackRegex           = regexp.MustCompile("^https?://open\\.spotify\\.com/(?:intl-[a-z]{2}/)?track/[a-zA-Z0-9]*$")
	spPlaylistRegex        = regexp.MustCompile("(?:https:\\/\\/)?open\\.spotify\\.com\\/playlist\\/([a-zA-Z0-9]+)(?:\\?si=([a-zA-Z0-9]*))?")
	ytVideoRegex           = regexp.MustCompile("^(https?://)?(www\\.)?(youtube\\.com|music\\.youtube\\.com)/watch.*$")
	ytPlaylistRegex        = regexp.MustCompile("^(https?://)?(www\\.)?(youtube\\.com|music\\.youtube\\.com)/playlist.*$")
	twitchStreamRegex      = regexp.MustCompile("https://(?:www\\.)?twitch\\.tv/([a-zA-Z0-9_]+)")
	skip                   = make(chan struct{}, 1)
	pause                  = make(chan struct{}, 1)
	isPaused               = make(chan bool)
	playbackPositionSignal = make(chan struct{}, 1)
	playbackPositionData   = make(chan time.Duration)
	autoKickState          bool
	autoPingState          bool
)

type ytTrack struct {
	Id              string
	title           string
	streamUrl       string
	duration        time.Duration
	author          string
	publishDate     time.Time
	video           *youtube.Video
	bitrate         int
	audiosamplerate string
	thumbnail       string
	format          *youtube.Format
}

func playHandler(s *discordgo.Session, i *discordgo.InteractionCreate, playNext bool) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		log.Println(err)
		return
	}

	user, err := s.User("@me")
	if err != nil {
		log.Println(err)
		return
	}
	avatarUrl := user.AvatarURL("")
	embed := &discordgo.MessageEmbed{
		Timestamp: timestamp(i.Interaction),
		Author:    &discordgo.MessageEmbedAuthor{Name: "MetalFistBot 6000", IconURL: avatarUrl},
		Footer:    &discordgo.MessageEmbedFooter{Text: fmt.Sprint("Requested by ", i.Member.User.Username)},
		Color:     botThemeColor,
	}

	guild, _ := s.State.Guild(i.GuildID)
	botVoiceState, err := s.State.VoiceState(guild.ID, botID)
	if err != nil && !errors.Is(err, discordgo.ErrStateNotFound) {
		fmt.Println(errors.Is(err, discordgo.ErrStateNotFound))
		log.Println("Error getting voice state", err)
		return
	}

	for _, vs := range guild.VoiceStates {
		if vs.UserID == i.Member.User.ID {
			for _, channel := range guild.Channels {
				if channel.ID == vs.ChannelID {
					bitrate = channel.Bitrate
				}
			}
			var requestedUnit string
			var err error
			connection, err = s.ChannelVoiceJoin(i.GuildID, vs.ChannelID, false, false)
			if err != nil {
				log.Println("Error joining vc:", err)
				return
			}

			if botVoiceState == nil {
				setSpeakingState(true)
				err := connection.Speaking(true)
				if err != nil {
					log.Println(err)
					return
				}
				options := dca.StdEncodeOptions
				options.BufferedFrames = 100
				options.FrameDuration = 20
				options.CompressionLevel = 5
				file, err := os.Open("./data/announcer.mp3")
				if err != nil {
					log.Println(err)
					return
				}
				reader, err := dca.EncodeMem(file, options)
				done := make(chan error)
				dca.NewStream(reader, connection, done)
				if err != nil {
					log.Println(err)
					return
				}
				<-done
				err = connection.Speaking(false)
				if err != nil {
					log.Println(err)
					return
				}
				setSpeakingState(false)
			}

			var videoUrl string
			if i.Type == discordgo.InteractionApplicationCommand {
				videoUrl = i.ApplicationCommandData().Options[0].StringValue()
			}
			if i.Type == discordgo.InteractionMessageComponent {
				videoUrl = i.MessageComponentData().Values[0]
			}

			switch {
			case ytVideoRegex.MatchString(videoUrl):
				requestedUnit = addTrack(videoUrl, playNext)
				break
			case ytPlaylistRegex.MatchString(videoUrl):
				requestedUnit = addPlaylist(videoUrl, playNext)
				break
			case spTrackRegex.MatchString(videoUrl):
				requestedUnit = spotifyTrackHandler(videoUrl, playNext)
				break
			case spPlaylistRegex.MatchString(videoUrl):
				requestedUnit = spotifyPlaylistHandler(videoUrl, playNext)
				break
			case twitchStreamRegex.MatchString(videoUrl):
				requestedUnit = twitchHandler(videoUrl, playNext, bitrate)
				break
			default:
				_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
					Content: "Please use a valid link for a video or a playlist",
				})
				if err != nil {
					log.Println(err)
					return
				}
			}

			if requestedUnit == "" {
				embed.Description = fmt.Sprint("Error playing audio")
				_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
					Embeds: []*discordgo.MessageEmbed{embed},
				})
				if err != nil {
					log.Println(err)
					return
				}
			} else {
				if !getSpeakingState() {
					embed.Description = fmt.Sprintf("Playing **%s** requested by %s", requestedUnit, i.Member.User.Username)
					_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
						Embeds: []*discordgo.MessageEmbed{embed},
					})
					if err != nil {
						log.Println(err)
						return
					}

					playQueue(s, i)
				} else {
					if playNext {
						embed.Description = fmt.Sprint("**", requestedUnit, "** is playing next, requested by ", i.Member.User.Username)
					} else {
						embed.Description = fmt.Sprint("**", requestedUnit, "** was added to the queue, by ", i.Member.User.Username)
					}
					_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
						Embeds: []*discordgo.MessageEmbed{embed},
					})
					if err != nil {
						log.Println(err)
						return
					}
				}
			}
			//connection.Disconnect()
			return
		}
	}

	_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: "You need to be in a voice channel to use this command!",
	})
	if err != nil {
		log.Println(err)
		return
	}
}

var commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
	"play": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		playHandler(s, i, false)
	},
	"play-next": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if !(len(queue) == 0) {
			playHandler(s, i, true)
		} else {
			playHandler(s, i, false)
		}
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
	"pause": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		pause <- struct{}{}
		user, err := s.User("@me")
		avatarUrl := user.AvatarURL("")
		embed := &discordgo.MessageEmbed{
			Timestamp: timestamp(i.Interaction),
			Author:    &discordgo.MessageEmbedAuthor{Name: "MetalFistBot 6000", IconURL: avatarUrl},
			Color:     botThemeColor,
			Footer:    &discordgo.MessageEmbedFooter{Text: fmt.Sprint("Requested by ", i.Member.User.Username)},
		}

		//fmt.Println("LEN:", len(pause), <-pause)

		if len(queue) == 0 {
			embed.Description = "There is no track currently playing"
			err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
			})
			if err != nil {
				log.Println(err)
			}
		} else {
			if <-isPaused {
				embed.Description = "Paused track"
				err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
				})
				if err != nil {
					log.Println(err)
				}
			} else {
				embed.Description = "Unpaused track"
				err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
				})
				if err != nil {
					log.Println(err)
				}
			}
		}
	},
	"shuffle": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		currentlyPlaying := queue[0]
		remainingQueue := queue[1:]

		for i := range remainingQueue {
			j := rand.Intn(len(remainingQueue) - 1)
			remainingQueue[i], remainingQueue[j] = remainingQueue[j], remainingQueue[i]
		}

		queue = append([]ytTrack{currentlyPlaying}, remainingQueue...)

		user, err := s.User("@me")
		avatarUrl := user.AvatarURL("")
		embed := &discordgo.MessageEmbed{
			Timestamp:   timestamp(i.Interaction),
			Author:      &discordgo.MessageEmbedAuthor{Name: "MetalFistBot 6000", IconURL: avatarUrl},
			Color:       botThemeColor,
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

		user, err := s.User("@me")
		avatarUrl := user.AvatarURL("")

		embed := &discordgo.MessageEmbed{
			Timestamp: timestamp(i.Interaction),
			Color:     botThemeColor,
			Author:    &discordgo.MessageEmbedAuthor{Name: "MetalFistBot 6000", IconURL: avatarUrl},
			Footer:    &discordgo.MessageEmbedFooter{Text: fmt.Sprint("Requested by ", i.Member.User.Username)},
		}
		if len(queue) <= 0 {
			embed.Description = "Queue is empty."
			_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Embeds: []*discordgo.MessageEmbed{embed},
			})
			if err != nil {
				return
			}
		} else {
			page := 1
			if i.ApplicationCommandData().Options != nil {
				page = int(i.ApplicationCommandData().Options[0].IntValue())
			}
			start := (page - 1) * 10
			end := page * 10
			if end > len(queue) {
				end = len(queue)
			}

			embed.Title = fmt.Sprint("Queue", page, "/", totalPages(queue))

			for i := start; i < end; i++ {
				embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
					Name:   strconv.Itoa(i + 1),
					Value:  fmt.Sprint(queue[i].title, " - ", formatDuration(queue[0].duration)),
					Inline: false,
				})
			}

			minSelected := 1
			pages := discordgo.SelectMenu{MinValues: &minSelected, MaxValues: 1, CustomID: "queuePage"}

			for j := 0; j < totalPages(queue); j++ {
				pages.Options = append(pages.Options, discordgo.SelectMenuOption{
					Label: fmt.Sprint("Page ", j+1),
					Value: strconv.Itoa(j + 1),
					Emoji: discordgo.ComponentEmoji{Name: ":queue_play_next_24dp_FILL0_wght4", ID: "1237399904846876692"},
				})
			}

			log.Println(i.Interaction.ID)

			_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Flags:  discordgo.MessageFlagsEphemeral,
				Embeds: []*discordgo.MessageEmbed{embed},
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{pages}},
				},
			})
			if err != nil {
				log.Println(err)
				return
			}

			prevInteraction := i.Interaction

			s.AddHandler(func(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
				if interaction.Type == discordgo.InteractionMessageComponent {
					if interaction.MessageComponentData().CustomID == "queuePage" {
						embed.Fields = nil
						page, _ := strconv.Atoi(interaction.MessageComponentData().Values[0])
						start := (page - 1) * 10
						end := page * 10
						if end > len(queue) {
							end = len(queue)
						}

						embed.Title = fmt.Sprint("Queue ", page, "/", totalPages(queue))

						for i := start; i < end; i++ {
							embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
								Name:   strconv.Itoa(i + 1),
								Value:  fmt.Sprint(queue[i].title, " - ", formatDuration(queue[0].duration)),
								Inline: false,
							})
						}

						err = s.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource,
							Data: &discordgo.InteractionResponseData{
								Embeds: []*discordgo.MessageEmbed{embed},
								Components: []discordgo.MessageComponent{
									discordgo.ActionsRow{Components: []discordgo.MessageComponent{pages}},
								},
							}})
						if err != nil {
							fmt.Println(err)
							return
						}

						err := s.InteractionResponseDelete(prevInteraction)
						if err != nil {
							fmt.Println(err)
							return
						}

						prevInteraction = interaction.Interaction
					}
				}
			})
		}

		/*s.AddHandlerOnce(func(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
				if interaction.Type == discordgo.InteractionMessageComponent {
					if interaction.MessageComponentData().CustomID == "queuePage" {
						embed.Fields = nil
						page, _ := strconv.Atoi(interaction.MessageComponentData().Values[0])
						start := (page - 1) * 10
						end := page * 10
						if end > len(queue) {
							end = len(queue)
						}

						for i := start; i < end; i++ {
							embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
								Name:   strconv.Itoa(i + 1),
								Value:  fmt.Sprint(queue[i].title, " - ", formatDuration(queue[0].duration)),
								Inline: false,
							})
						}

						updatedResponse := &discordgo.WebhookEdit{
							Embeds: &[]*discordgo.MessageEmbed{embed},
							Components: &[]discordgo.MessageComponent{
								discordgo.ActionsRow{Components: []discordgo.MessageComponent{pages}},
							},
						}
						responseEdit, err := session.InteractionResponseEdit(i.Interaction, updatedResponse)
						fmt.Println(responseEdit)
						if err != nil {
							fmt.Println(err)
							return
						}
					}
				}
			})

		}*/
	},
	"nowplaying": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if len(queue) != 0 {
			playbackPositionSignal <- struct{}{}
			user, err := s.User("@me")
			avatarUrl := user.AvatarURL("")
			if err != nil {
				log.Println(err)
				return
			}
			var position = <-playbackPositionData
			var totalTrackDuration = queue[0].duration
			var totalTrackDurationPretty = formatDuration(totalTrackDuration)
			var playbackPositionPretty = formatDuration(position)
			var progressPercentage = (position.Seconds() / totalTrackDuration.Seconds()) * 100
			var filledUnits = int(math.Round((progressPercentage / 100) * progressBarUnits))
			var progressBar string = ""
			for range filledUnits {
				progressBar = progressBar + "█"
			}
			for range progressBarUnits - filledUnits {
				progressBar = progressBar + "░"
			}

			embed := &discordgo.MessageEmbed{
				Timestamp:   timestamp(i.Interaction),
				Author:      &discordgo.MessageEmbedAuthor{Name: "MetalFistBot 6000", IconURL: avatarUrl},
				Title:       queue[0].title,
				Thumbnail:   &discordgo.MessageEmbedThumbnail{URL: queue[0].thumbnail},
				URL:         "https://music.youtube.com/watch?v=" + queue[0].Id,
				Description: fmt.Sprint(queue[0].author, "\n", progressBar, " - ", playbackPositionPretty, " / ", totalTrackDurationPretty),
				Footer:      &discordgo.MessageEmbedFooter{Text: fmt.Sprint("Requested by ", i.Member.User.Username)},
				Color:       botThemeColor,
			}

			err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
			})
			if err != nil {
				log.Println(err)
				return
			}
		}
	},
	"leave": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		user, err := s.User("@me")
		avatarUrl := user.AvatarURL("")
		if err != nil {
			log.Println(err)
			return
		}

		embed := &discordgo.MessageEmbed{
			Timestamp: timestamp(i.Interaction),
			Author:    &discordgo.MessageEmbedAuthor{Name: "MetalFistBot 6000", IconURL: avatarUrl},
			Footer:    &discordgo.MessageEmbedFooter{Text: fmt.Sprint("Requested by ", i.Member.User.Username)},
			Color:     botThemeColor,
		}

		if connection != nil && botInVoiceChannel(s, i) {
			setSpeakingState(false)
			err := connection.Speaking(false)
			if err != nil {
				log.Println(err)
				return
			}
			err = connection.Disconnect()
			if err != nil {
				log.Println("could not disconnect:", err)
				return
			}
			connection = nil
			queue = nil

			embed.Description = "Successfully left the channel"
			err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
			})
			if err != nil {
				log.Println(err)
				return
			}
		} else {
			embed.Description = "I am currently not connected to a channel"
			err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
			})
			if err != nil {
				log.Println(err)
				return
			}
		}
	},
	"skip": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		user, err := s.User("@me")
		avatarUrl := user.AvatarURL("")

		embed := &discordgo.MessageEmbed{
			Timestamp: timestamp(i.Interaction),
			Author:    &discordgo.MessageEmbedAuthor{Name: "MetalFistBot 6000", IconURL: avatarUrl},
			Footer:    &discordgo.MessageEmbedFooter{Text: fmt.Sprint("Requested by ", i.Member.User.Username)},
			Color:     botThemeColor,
		}
		if len(queue) == 0 {
			embed.Description = "There are no tracks currently playing."
			err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
			})
			if err != nil {
				log.Println(err)
				return
			}
			return
		}

		log.Println(queue)

		if len(queue) > 1 { // Current playing track is still in the array, thus check if there are more than 1
			skip <- struct{}{}
			embed.Description = ":fast_forward: Track skipped!"
			err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
			})
			if err != nil {
				log.Println("Error sending message:", err)
			}
		} else {
			embed.Description = "There are no more tracks in the queue."
			err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
			})
			if err != nil {
				log.Println(err)
				return
			}
		}

	},
	"playlists": func(s *discordgo.Session, i *discordgo.InteractionCreate) {

		const desc = `* [Phonk](https://music.youtube.com/playlist?list=PL9fJVbkciwbfFTQOQ8ykmKlORtfPhct0V&si=diR29jeR6JIvQ4Wr)
* [Lyrische Meisterwerke](https://music.youtube.com/playlist?list=PL9fJVbkciwbfxF-1Eu7UqDEsxpoZgWxF0&si=9svtyy4nG9XODAf2)
* [Mainly Hardstyle](https://music.youtube.com/playlist?list=PL9fJVbkciwbdKciqQUzx2XFxnaQsQfKYF&si=_HFamWDe9k9cQA2i)
* [Frenchcore](https://music.youtube.com/playlist?list=PL9fJVbkciwbed6rQkv00xCZ-Wy-MeEPQP&si=U2DVK4hm2dWDEy1F)
* [Hardtekk](https://music.youtube.com/playlist?list=PL9fJVbkciwbdzTL5UgrjMvhWGi3RT8NBV&si=qA5uJmLIXWXIo_cr)
* [Deutsche Tekke](https://music.youtube.com/playlist?list=PL9fJVbkciwbfOq3yEgFzWXzLusFAr199d&si=BGG7KPMbmP3zKpB7)
* [Banger](https://music.youtube.com/playlist?list=PL9fJVbkciwbf3VoxokMwOAr0gwSHG9b7T&si=eAypOSD_vtRCVt8v)
* [Techno](https://music.youtube.com/playlist?list=PL9fJVbkciwbdlb74QY7Wec27zMsRF5M41)
* [Hypertechno](https://music.youtube.com/playlist?list=PL9fJVbkciwbdA2XdLitadlBczKoaqaC8i&si=9VXe0MhXVoChlqWE)
* [Nightcore](https://music.youtube.com/playlist?list=PL9fJVbkciwbdJfUS6c7XL62qfjGzaabxS&si=tJaNr7Wr6AnIfHWk)`

		user, err := s.User("@me")
		avatarUrl := user.AvatarURL("")
		if err != nil {
			log.Println(err)
			return
		}

		embed := &discordgo.MessageEmbed{
			Title:       "Playlists",
			Description: desc,
			Timestamp:   timestamp(i.Interaction),
			Author:      &discordgo.MessageEmbedAuthor{Name: "MetalFistBot 6000", IconURL: avatarUrl},
			Footer:      &discordgo.MessageEmbedFooter{Text: fmt.Sprint("Requested by ", i.Member.User.Username)},
			Color:       botThemeColor,
		}
		minSelect := 1
		menu := discordgo.SelectMenu{MinValues: &minSelect, MaxValues: 1,
			Options: []discordgo.SelectMenuOption{
				{
					Label: "Phonk",
					Value: "https://music.youtube.com/playlist?list=PL9fJVbkciwbfFTQOQ8ykmKlORtfPhct0V&si=diR29jeR6JIvQ4Wr",
					Emoji: discordgo.ComponentEmoji{Name: ":queue_music_24dp_FILL0_wght400_G:", ID: "1237400059335802961"},
				},
				{
					Label: "Lyrische Meisterwerke",
					Value: "https://music.youtube.com/playlist?list=PL9fJVbkciwbfxF-1Eu7UqDEsxpoZgWxF0&si=9svtyy4nG9XODAf2",
					Emoji: discordgo.ComponentEmoji{Name: ":queue_music_24dp_FILL0_wght400_G:", ID: "1237400059335802961"},
				},
				{
					Label: "Mainly Hardstyle",
					Value: "https://music.youtube.com/playlist?list=PL9fJVbkciwbdKciqQUzx2XFxnaQsQfKYF&si=_HFamWDe9k9cQA2i",
					Emoji: discordgo.ComponentEmoji{Name: ":queue_music_24dp_FILL0_wght400_G:", ID: "1237400059335802961"},
				},
				{
					Label: "Frenchcore",
					Value: "https://music.youtube.com/playlist?list=PL9fJVbkciwbed6rQkv00xCZ-Wy-MeEPQP&si=U2DVK4hm2dWDEy1F",
					Emoji: discordgo.ComponentEmoji{Name: ":queue_music_24dp_FILL0_wght400_G:", ID: "1237400059335802961"},
				},
				{
					Label: "Hardtekk",
					Value: "https://music.youtube.com/playlist?list=PL9fJVbkciwbdzTL5UgrjMvhWGi3RT8NBV&si=qA5uJmLIXWXIo_cr",
					Emoji: discordgo.ComponentEmoji{Name: ":queue_music_24dp_FILL0_wght400_G:", ID: "1237400059335802961"},
				},
				{
					Label: "Deutsche Tekke",
					Value: "https://music.youtube.com/playlist?list=PL9fJVbkciwbfOq3yEgFzWXzLusFAr199d&si=BGG7KPMbmP3zKpB7",
					Emoji: discordgo.ComponentEmoji{Name: ":queue_music_24dp_FILL0_wght400_G:", ID: "1237400059335802961"},
				},
				{
					Label: "Banger",
					Value: "https://music.youtube.com/playlist?list=PL9fJVbkciwbf3VoxokMwOAr0gwSHG9b7T&si=eAypOSD_vtRCVt8v",
					Emoji: discordgo.ComponentEmoji{Name: ":queue_music_24dp_FILL0_wght400_G:", ID: "1237400059335802961"},
				},
				{
					Label: "Techno",
					Value: "https://music.youtube.com/playlist?list=PL9fJVbkciwbdlb74QY7Wec27zMsRF5M41",
					Emoji: discordgo.ComponentEmoji{Name: ":queue_music_24dp_FILL0_wght400_G:", ID: "1237400059335802961"},
				},
				{
					Label: "Hypertechno",
					Value: "https://music.youtube.com/playlist?list=PL9fJVbkciwbdA2XdLitadlBczKoaqaC8i&si=9VXe0MhXVoChlqWE",
					Emoji: discordgo.ComponentEmoji{Name: ":queue_music_24dp_FILL0_wght400_G:", ID: "1237400059335802961"},
				},
				{
					Label: "Nightcore",
					Value: "https://music.youtube.com/playlist?list=PL9fJVbkciwbdJfUS6c7XL62qfjGzaabxS&si=tJaNr7Wr6AnIfHWk",
					Emoji: discordgo.ComponentEmoji{Name: ":queue_music_24dp_FILL0_wght400_G:", ID: "1237400059335802961"},
				},
			},
			CustomID: "playlistSelect",
		}

		actionRow := discordgo.ActionsRow{Components: []discordgo.MessageComponent{menu}}

		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}, Components: []discordgo.MessageComponent{actionRow}},
		})
		if err != nil {
			log.Println(err)
			return
		}
	},
	"help": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		user, err := s.User("@me")
		avatarUrl := user.AvatarURL("")
		if err != nil {
			log.Println(err)
			return
		}

		fields := []*discordgo.MessageEmbedField{{Name: "/play", Value: "plays or queues a song. options: url, the URL of the song to be played"}, {Name: "/shuffle", Value: "randomizes the queue"}, {Name: "/leave", Value: "cleans the bot and disconnects it from the voice channel"}, {Name: "/pause", Value: "pauses the current playing song. When already paused, then the command will unpause"}, {Name: "/nowplaying", Value: "gives some information about the playing song"}, {Name: "/skip", Value: "skips the playing song and plays the next from the queue"}, {Name: "/queue", Value: "shows the music queue of the bot. options: page, which page of the queue you want to see"}, {Name: "/autoplay", Value: "when enabled, the bot will play other songs when the queue ended"}}
		embed := &discordgo.MessageEmbed{
			Title:     "Help",
			Timestamp: timestamp(i.Interaction),
			Author:    &discordgo.MessageEmbedAuthor{Name: "MetalFistBot 6000", IconURL: avatarUrl},
			Footer:    &discordgo.MessageEmbedFooter{Text: fmt.Sprint("Requested by ", i.Member.User.Username)},
			Color:     botThemeColor,
			Fields:    fields,
		}

		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
		})
		if err != nil {
			log.Println(err)
			return
		}
	},
	"logs": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		user, err := s.User("@me")
		avatarUrl := user.AvatarURL("")
		if err != nil {
			log.Println(err)
			return
		}
		embed := &discordgo.MessageEmbed{
			Title:     "Logs",
			Timestamp: timestamp(i.Interaction),
			Author:    &discordgo.MessageEmbedAuthor{Name: "MetalFistBot 6000", IconURL: avatarUrl},
			Footer:    &discordgo.MessageEmbedFooter{Text: fmt.Sprint("Requested by ", i.Member.User.Username)},
			Color:     botThemeColor,
		}

		if i.Member.User.ID == botOwnerID {

			lastLines, err := readLastLines()
			if err != nil {
				fmt.Println("Error:", err)
				return
			}
			embed.Description = lastLines
			err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
			})
			if err != nil {
				log.Println(err)
				return
			}
		} else {
			embed.Description = "You are not authorized to use this command!"
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
			})
			if err != nil {
				log.Println(err)
				return
			}
		}
	},
	"auto-ping": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		autoPingState = !autoPingState

		if autoPingState {
			if len(i.ApplicationCommandData().Options) <= 0 {
				err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{Content: "please specify a user"},
				})
				if err != nil {
					log.Println(err)
					return
				}
				return
			}
			user := i.ApplicationCommandData().Options[0].UserValue(s)

			go pingingInstance(s, i.ChannelID, user.ID)
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Content: "Auto-ping enabled"},
			})
			if err != nil {
				log.Println(err)
				return
			}
		} else {
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Content: "Auto-ping disabled"},
			})
			if err != nil {
				log.Println(err)
				return
			}
		}
	},
	"auto-kick": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		autoKickState = !autoKickState
		guild, err := s.State.Guild(i.GuildID)
		if err != nil {
			fmt.Println("Error retrieving guild:", err)
			return
		}

		if autoKickState {
			for _, vs := range guild.VoiceStates {
				if vs.UserID == userToKick {
					vcu, err := s.State.VoiceState(i.GuildID, i.Member.User.ID)
					if err != nil {
						fmt.Println("Error getting voice state:", err)
						return
					}

					if vs.UserID == userToKick && vs.ChannelID == vcu.ChannelID {
						err := s.GuildMemberMove(vs.GuildID, vs.UserID, nil)
						if err != nil {
							log.Println("Error kicking user:", err)
						} else {
							log.Println("Kicked user", vs.UserID, "from the voice channel")
						}
					}
				}
			}
		}

		if autoKickState {
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Content: "Auto-kick enabled"},
			})
			if err != nil {
				log.Println(err)
				return
			}
		} else {
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Content: "Auto-kick disabled"},
			})
			if err != nil {
				log.Println(err)
				return
			}
		}

	},
	"quote-add": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		name := i.ApplicationCommandData().Options[0].StringValue()
		quote := i.ApplicationCommandData().Options[1].StringValue()
		user := i.Member.User.ID
		db := DBConnection()

		_, err := db.Exec("insert into quote (name, quote, author) values (?, ?, ?)", name, quote, user)
		if err != nil {
			log.Println(err)
			return
		}

		defer func(db *sql.DB) {
			err := db.Close()
			if err != nil {
				log.Println(err)
			}
		}(db)

		bot, err := s.User("@me")
		avatarUrl := bot.AvatarURL("")
		if err != nil {
			log.Println(err)
			return
		}
		embed := &discordgo.MessageEmbed{
			Title:       "Quote",
			Timestamp:   timestamp(i.Interaction),
			Description: fmt.Sprintf("### Added: \n**Name:** %s\n**Quote:** \"%s\"", name, quote),
			Author:      &discordgo.MessageEmbedAuthor{Name: "MetalFistBot 6000", IconURL: avatarUrl},
			Footer:      &discordgo.MessageEmbedFooter{Text: fmt.Sprint("Requested by ", i.Member.User.Username)},
			Color:       botThemeColor,
		}
		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
		})
		if err != nil {
			log.Println(err)
			return
		}
	},
	"quote-delete": func(s *discordgo.Session, i *discordgo.InteractionCreate) {

	},
	"quote": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		db := DBConnection()
		var quote, name, author string

		err := db.QueryRow("SELECT quote, name, author FROM quote ORDER BY RANDOM()").Scan(&quote, &name, &author)
		if err != nil {
			log.Println(err)
			return
		}

		user, err := s.User(author)
		if err != nil {
			log.Println("Couldn't retrieve user:", err)
			return
		}

		bot, err := s.User("@me")
		avatarUrl := bot.AvatarURL("")
		if err != nil {
			log.Println(err)
			return
		}
		embed := &discordgo.MessageEmbed{
			Title:       "Quote",
			Timestamp:   timestamp(i.Interaction),
			Description: "### " + name + "\n \"" + quote + "\"\n from **" + user.Username + "**",
			Author:      &discordgo.MessageEmbedAuthor{Name: "MetalFistBot 6000", IconURL: avatarUrl},
			Footer:      &discordgo.MessageEmbedFooter{Text: fmt.Sprint("Requested by ", i.Member.User.Username)},
			Color:       botThemeColor,
		}
		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
		})
		if err != nil {
			log.Println(err)
			return
		}
	},
	"quote-list": func(s *discordgo.Session, i *discordgo.InteractionCreate) {

	},
}
