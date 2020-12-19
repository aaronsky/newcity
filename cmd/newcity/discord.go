package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/session"
)

var (
	// ErrNoDiscordBotToken happens when a token is not provided via the -bot_token flag or the BOT_TOKEN environment variable.
	ErrNoDiscordBotToken = errors.New("no Discord bot token provided")
	// ErrNoDiscordChannelID happens when a channel ID is not provided via the -channel_id flag, or it is not a valid int64.
	ErrNoDiscordChannelID = errors.New("no Discord channel ID provided")
)

// PostToDiscord posts the given messages to Discord.
func PostToDiscord(messages ...string) error {
	if len(messages) == 0 {
		log.Println(warn("no messages to send to Discord"))
		return nil
	}

	token := *botTokenFlag
	if token == "" {
		token = os.Getenv("BOT_TOKEN")
		if token == "" {
			return ErrNoDiscordBotToken
		}
	}

	channelID := *channelIDFlag
	if channelID == 0 {
		return ErrNoDiscordChannelID
	}

	// auth with Discord
	s, err := session.New("Bot " + token)
	if err != nil {
		return err
	}

	// get channel to send to
	if err := s.Open(); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	defer Close(s)

	// create message
	for _, message := range messages {
		m, err := s.SendText(discord.ChannelID(channelID), message)
		if err != nil {
			return err
		}

		log.Println("SENT:", m.ID)
	}

	return nil
}
