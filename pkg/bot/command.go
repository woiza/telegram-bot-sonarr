package bot

import (
	"fmt"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/woiza/telegram-bot-sonarr/pkg/utils"
	"golift.io/starr"
	"golift.io/starr/sonarr"
)

func (b *Bot) handleCommand(update tgbotapi.Update, s *sonarr.Sonarr) {

	userID, err := b.getUserID(update)
	if err != nil {
		fmt.Printf("Cannot handle command: %v", err)
		return
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")

	switch update.Message.Command() {

	case "q", "query", "add", "Q", "Query", "Add":
		b.setActiveCommand(userID, AddSeriesCommand)
		b.processAddCommand(update, userID, s)

	case "series", "library", "l":
		b.setActiveCommand(userID, LibraryMenuCommand)
		b.processLibraryCommand(update, userID, s)

	case "delete", "remove", "Delete", "Remove", "d":
		b.setActiveCommand(userID, DeleteSeriesCommand)
		b.processDeleteCommand(update, userID, s)

	case "clear", "cancel", "stop":
		b.clearState(update)
		msg.Text = "All commands have been cleared"
		b.sendMessage(msg)

	case "diskspace", "disk", "free", "rootfolder", "rootfolders":
		rootFolders, err := s.GetRootFolders()
		if err != nil {
			msg.Text = err.Error()
			fmt.Println(err)
			b.sendMessage(msg)
			break
		}
		msg.Text = utils.PrepareRootFolders(rootFolders)
		msg.ParseMode = "MarkdownV2"
		msg.DisableWebPagePreview = true
		b.sendMessage(msg)

	case "up", "upcoming":
		calendar := sonarr.Calendar{
			Start:       time.Now(),
			End:         time.Now().AddDate(0, 0, 30), // 30 days
			Unmonitored: *starr.True(),
		}
		upcoming, err := s.GetCalendar(calendar)
		if err != nil {
			msg.Text = err.Error()
			fmt.Println(err)
			b.sendMessage(msg)
			break
		}
		if len(upcoming) == 0 {
			msg.Text = "no upcoming releases in the next 30 days"
			b.sendMessage(msg)
			break
		}
		b.sendUpcoming(upcoming, &msg)

	case "rss", "RSS":
		command := sonarr.CommandRequest{
			Name:      "RssSync",
			SeriesIDs: []int64{},
		}
		_, err := s.SendCommand(&command)
		if err != nil {
			msg.Text = err.Error()
			fmt.Println(err)
			b.sendMessage(msg)
			break
		}
		msg.Text = "RSS sync started"
		b.sendMessage(msg)

	case "searchmonitored":
		series, err := s.GetSeries(0)
		if err != nil {
			msg.Text = err.Error()
			fmt.Println(err)
			b.sendMessage(msg)
			break
		}
		var monitoredSeriesIDs []int64
		for _, series := range series {
			if series.Monitored {
				monitoredSeriesIDs = append(monitoredSeriesIDs, series.ID)
			}
		}
		command := sonarr.CommandRequest{
			Name:      "SeriesSearch",
			SeriesIDs: monitoredSeriesIDs,
		}
		_, err = s.SendCommand(&command)
		if err != nil {
			msg.Text = err.Error()
			fmt.Println(err)
			b.sendMessage(msg)
			break
		}
		msg.Text = "Search for monitored series started"
		b.sendMessage(msg)

	case "updateAll", "updateall":
		series, err := s.GetSeries(0)
		if err != nil {
			msg.Text = err.Error()
			fmt.Println(err)
			b.sendMessage(msg)
			break
		}
		var allSeriesIDs []int64
		for _, series := range series {
			allSeriesIDs = append(allSeriesIDs, series.ID)
		}
		command := sonarr.CommandRequest{
			Name:      "RefresSeries",
			SeriesIDs: allSeriesIDs,
		}
		_, err = s.SendCommand(&command)
		if err != nil {
			msg.Text = err.Error()
			fmt.Println(err)
			b.sendMessage(msg)
			break
		}
		msg.Text = "Update All started"
		b.sendMessage(msg)

	case "system", "System", "systemstatus", "Systemstatus":
		status, err := s.GetSystemStatus()
		if err != nil {
			msg.Text = err.Error()
			fmt.Println(err)
			b.sendMessage(msg)
			break
		}
		message := prettyPrint(status)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, message)
		b.sendMessage(msg)

	case "getid", "id":
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Your user ID: %d", userID))
		b.sendMessage(msg)

	default:
		msg.Text = fmt.Sprintf("Hello %v!\n", update.Message.From)
		msg.Text += "Here's a list of commands at your disposal:\n\n"
		msg.Text += "/q [series] - searches a series \n"
		msg.Text += "/library [series] - manage series(s)\n"
		msg.Text += "/delete [series] - deletes a series\n"
		msg.Text += "/clear - deletes all sent commands\n"
		msg.Text += "/free  - lists free disk space \n"
		msg.Text += "/up\t\t\t\t - lists upcoming episodes in the next 30 days\n"
		msg.Text += "/rss \t\t - performs a RSS sync\n"
		msg.Text += "/searchmonitored - searches all monitored series\n"
		msg.Text += "/updateall - updates metadata and rescans files/folders\n"
		msg.Text += "/system - shows your Sonarr configuration\n"
		msg.Text += "/id - shows your Telegram user ID"
		b.sendMessage(msg)
	}
}
