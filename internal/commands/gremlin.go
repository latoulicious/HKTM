package commands

import (
	"math/rand"

	"github.com/bwmarrin/discordgo"
	"github.com/latoulicious/HKTM/pkg/logging"
)

// gremlinImages contains the list of image URLs for the gremlin command
var gremlinImages = []string{
	"https://cdn.discordapp.com/attachments/1119291447926075412/1402499775227756705/aston_machan_umamusume_and_1_more_drawn_by_bacius__ecfb7d4ef0e7cb8c7ae0f534b10ee3864.jpg?ex=68942333&is=6892d1b3&hm=4419e0f33ba03920ec5ff6a5f2943f4c4f571249b420fb97503413706c854680&",
	"https://cdn.discordapp.com/attachments/1119291447926075412/1402499774980030546/aston_machan_umamusume_and_1_more_drawn_by_bacius__ecfb7d4ef0e7cb8c7ae0f534b10ee3863.jpg?ex=68942333&is=6892d1b3&hm=34457c312dccaec3bc46c8d048ccc14cc2f8b74fc236b49d706e628b91bfcb6e&",
	// Add more image URLs here as needed
}

// gremlinFooters contains random footer messages for the gremlin command
var gremlinFooters = []string{
	"You must absolutely follow these three rules, okay.",
	"Maachan's gremlin energy is unstoppable!",
	"Random Maachan moment activated!",
	"The gremlin has been summoned!",
	"Maachan chaos incoming!",
	"Warning: Gremlin detected!",
	"Maachan's mischievous side appears!",
	// Add more footer messages here as needed
}

// GremlinCommand sends a random image from the gremlin collection
func GremlinCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Initialize centralized logging for this command
	loggerFactory := logging.GetGlobalLoggerFactory()
	logger := loggerFactory.CreateCommandLogger("gremlin")
	logger.Info("Gremlin command executed", map[string]interface{}{
		"user_id":    m.Author.ID,
		"username":   m.Author.Username,
		"guild_id":   m.GuildID,
		"channel_id": m.ChannelID,
	})
	if len(gremlinImages) == 0 {
		logger.Error("No gremlin images available", nil, map[string]interface{}{
			"user_id":  m.Author.ID,
			"guild_id": m.GuildID,
		})
		s.ChannelMessageSend(m.ChannelID, "No Maachan gremlin images available!")
		return
	}

	// Select a random image from the collection
	randomImage := gremlinImages[rand.Intn(len(gremlinImages))]

	// Select a random footer message
	randomFooter := gremlinFooters[rand.Intn(len(gremlinFooters))]

	// Create an embed with the random image and footer
	embed := &discordgo.MessageEmbed{
		Color: 0x9932cc, // Purple color
		Image: &discordgo.MessageEmbedImage{
			URL: randomImage,
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: randomFooter,
		},
	}

	_, err := s.ChannelMessageSendEmbed(m.ChannelID, embed)
	if err != nil {
		logger.Error("Failed to send gremlin embed", err, map[string]interface{}{
			"channel_id": m.ChannelID,
			"guild_id":   m.GuildID,
		})
	} else {
		logger.Debug("Gremlin image sent successfully", map[string]interface{}{
			"user_id":         m.Author.ID,
			"guild_id":        m.GuildID,
			"selected_image":  randomImage,
			"selected_footer": randomFooter,
		})
	}
}
