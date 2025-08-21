package commands

import (
	"fmt"
	"runtime"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/latoulicious/HKTM/internal/version"
	"github.com/latoulicious/HKTM/pkg/logging"
)

var startTime = time.Now()

func AboutCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	loggerFactory := logging.GetGlobalLoggerFactory()
	logger := loggerFactory.CreateCommandLogger("about")
	logger.Info("About command executed", map[string]interface{}{
		"user_id":    m.Author.ID,
		"username":   m.Author.Username,
		"guild_id":   m.GuildID,
		"channel_id": m.ChannelID,
	})

	uptime := time.Since(startTime)
	uptimeStr := formatUptime(uptime)

	// memory usage
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	memoryUsage := fmt.Sprintf("%.2f MB", float64(memStats.Alloc)/1024/1024)

	// fetch build/version info
	info := version.Get()
	buildTime := info.BuildTime
	if t, err := time.Parse(time.RFC3339, info.BuildTime); err == nil {
		buildTime = t.UTC().Format("02 Jan 2006 15:04 UTC")
	}
	commitURL := fmt.Sprintf("https://github.com/latoulicious/HKTM/commit/%s", info.GitCommit)

	embed := &discordgo.MessageEmbed{
		Title:       "Bot Information",
		Description: "Tomakomai's Tourism Ambassador!★",
		Color:       0x00ff00,
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Created and maintained by latoulicious",
		},
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Bot Name", Value: "Hokko Tarumae", Inline: true},
			{Name: "Version", Value: code(info.Version), Inline: true},
			{Name: "Commit", Value: fmt.Sprintf("[%s](%s)", info.ShortCommit, commitURL), Inline: true},
			{Name: "Repository", Value: "[GitHub](https://github.com/latoulicious/HKTM)", Inline: true},
			{Name: "Uptime", Value: uptimeStr, Inline: true},
			{Name: "Memory Usage", Value: memoryUsage, Inline: true},
			{Name: "Goroutines", Value: fmt.Sprintf("%d", runtime.NumGoroutine()), Inline: true},
			{Name: "Go Version", Value: runtime.Version(), Inline: true},
			{Name: "Platform", Value: fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH), Inline: true},
			{Name: "Build Time", Value: buildTime, Inline: true},
			{Name: "Ping", Value: fmt.Sprintf("%dms", s.HeartbeatLatency().Milliseconds()), Inline: true},
		},
		Image: &discordgo.MessageEmbedImage{
			URL: "https://c.tenor.com/ct99YJIYdvgAAAAC/tenor.gif",
		},
	}

	_, err := s.ChannelMessageSendEmbed(m.ChannelID, embed)
	if err != nil {
		logger.Error("Failed to send about embed", err, map[string]interface{}{
			"channel_id": m.ChannelID,
			"guild_id":   m.GuildID,
		})
	}
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
