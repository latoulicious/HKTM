package commands

import (
	"fmt"
	"runtime"
	"time"

	"github.com/bwmarrin/discordgo"
)

var startTime = time.Now()

// AboutCommand displays bot information including version, uptime, memory usage, and Go version
func AboutCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Calculate uptime
	uptime := time.Since(startTime)
	uptimeStr := formatUptime(uptime)

	// Get memory statistics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	memoryUsage := fmt.Sprintf("%.2f MB", float64(memStats.Alloc)/1024/1024)

	// Create embed
	embed := &discordgo.MessageEmbed{
		Title:       "Bot Information",
		Description: "Tomakomai's Tourism Ambassador!★",
		Color:       0x00ff00, // Green color
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Created by latoulicious | 2025",
			// IconURL: "https://cdn.discordapp.com/emojis/1198008186138021888.webp?size=96",
		},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Bot Name & Version",
				Value:  "Hokko Tarumae v1.0.0",
				Inline: true,
			},
			{
				Name:   "Uptime",
				Value:  uptimeStr,
				Inline: true,
			},
			{
				Name:   "Memory Usage",
				Value:  memoryUsage,
				Inline: true,
			},
			{
				Name:   "Go Version",
				Value:  runtime.Version(),
				Inline: true,
			},
			{
				Name:   "Platform",
				Value:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
				Inline: true,
			},
			{
				Name:   "Goroutines",
				Value:  fmt.Sprintf("%d", runtime.NumGoroutine()),
				Inline: true,
			},
		},
		Image: &discordgo.MessageEmbedImage{
			URL: "https://c.tenor.com/ct99YJIYdvgAAAAC/tenor.gif",
		},
	}

	s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

// formatUptime formats the uptime duration into a human-readable string
func formatUptime(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, seconds)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	} else {
		return fmt.Sprintf("%ds", seconds)
	}
}
