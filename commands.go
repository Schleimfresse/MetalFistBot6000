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
					Type:        discordgo.ApplicationCommandOptionInteger,
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
			Name:        "help",
			Description: "gives you an help interface",
		},
		{
			Name:        "logs",
			Description: "logs out the logs from the bot",
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
		{
			Name:        "auto-kick",
			Description: "have some fun",
		},
		{
			Name:        "auto-ping",
			Description: "have some fun",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "user",
					Description: "select the user which should get pinged",
					Required:    false,
				},
			},
		},
		{
			Name:        "quote-add",
			Description: "add a quote",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "name",
					Description: "name of the author of the quote",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "quote",
					Description: "the quote which should get added",
					Required:    true,
				},
			},
		},
		{
			Name:        "quote-remove",
			Description: "remove a quote",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "name",
					Description: "short name of the quote which should get removed",
					Required:    true,
				},
			},
		},
		{
			Name:        "quote",
			Description: "get a random quote",
		},
		{
			Name:        "quote-list",
			Description: "list of all quotes",
		},
	}
)
