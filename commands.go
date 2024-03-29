package main

import "github.com/bwmarrin/discordgo"

var (
	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "ping",
			Description: "measure the latency of the bot",
		},
		{
			Name:        "play-next",
			Description: "adds a song to the queue and plays it after the currently playing song",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "url",
					Description: "url of the track you want to play",
					Required:    true,
				},
			},
		},
		{
			Name:        "play",
			Description: "play a track",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "url",
					Description: "url of the track you want to play",
					Required:    true,
				},
			},
		},
		{
			Name:        "queue",
			Description: "take a look at the queue",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "page",
					Description: "page of the queue",
					Required:    false,
				},
			},
		},
		{
			Name:        "pause",
			Description: "pause the current played song",
		},
		{
			Name:        "nowplaying",
			Description: "gives detailed information of the played track",
		},
		{
			Name:        "leave",
			Description: "disconnects the bot from the voice channel",
		},
		{
			Name:        "skip",
			Description: "skip the current track",
		},
		{
			Name:        "shuffle",
			Description: "shuffle the queue",
		},
		{
			Name:        "playlists",
			Description: "Gives you the playlist list",
		},
	}
)
