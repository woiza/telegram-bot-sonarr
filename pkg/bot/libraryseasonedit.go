package bot

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/woiza/telegram-bot-sonarr/pkg/utils"
	"golift.io/starr"
	"golift.io/starr/sonarr"
)

const (
	LibrarySeasonEditToggleMonitor = "LIBRARY_SEASON_EDIT_TOGGLE_MONITOR"
	LibrarySeasonEditSubmitChanges = "LIBRARY_SEASON_EDIT_SUBMIT_CHANGES"
	LibrarySeasonEditGoBack        = "LIBRARY_SEASON_EDIT_GOBACK"
	LibrarySeasonEditCancel        = "LIBRARY_SEASON_EDIT_CANCEL"
	LibrarySeasonMonitor           = "LIBRARY_SEASON_MONITOR"
	LibrarySeasonUnmonitor         = "LIBRARY_SEASON_UNMONITOR"
	LibrarySeasonSearch            = "LIBRARY_SEASON_SEARCH"
	LibrarySeasonMonitorSearchNow  = "LIBRARY_SEASON_MONITOR_SEARCH_NOW"
	LibrarySeasonDelete            = "LIBRARY_SEASON_DELETE"
	LibrarySeasonGoBack            = "LIBRARY_SEASON_GOBACK"
)

func (b *Bot) librarySeasonEdit(update tgbotapi.Update) bool {
	userID, err := b.getUserID(update)
	if err != nil {
		fmt.Printf("Cannot manage library: %v", err)
		return false
	}

	command, exists := b.getLibraryState(userID)
	if !exists {
		return false
	}
	switch update.CallbackQuery.Data {
	case LibrarySeasonEditToggleMonitor:
		return b.handleLibrarySeasonEditToggleMonitor(command)
	case LibrarySeasonEditSubmitChanges:
		return b.handleLibrarySeasonEditSubmitChanges(update, command)
	case LibrarySeasonEditGoBack:
		b.setActiveCommand(userID, LibraryFilteredActive)
		b.setLibraryState(command.chatID, command)
		return b.showLibrarySeriesDetail(update, command)
	case LibrarySeasonEditCancel:
		b.clearState(update)
		b.sendMessageWithEdit(command, CommandsCleared)
		return false
	case LibrarySeasonMonitor:
		return b.handleLibrarySeriesSeasonMonitor(command)
	case LibrarySeasonUnmonitor:
		return b.handleLibrarySeriesSeasonUnmonitor(command)
	case LibrarySeasonSearch:
		return b.handleLibrarySeriesSeasonSearch(command)
	case LibrarySeasonMonitorSearchNow:
		return b.handleLibrarySeriesSeasonMonitorSearchNow(command)
	case LibrarySeasonDelete:
		return b.handleLibrarySeasonDeleteSeason(command)
	default:
		// Check if it starts with "SEASON_"
		if strings.HasPrefix(update.CallbackQuery.Data, "SEASON_") {
			return b.handleLibrarySeasonEditSelectSeason(update, command)
		}
		return b.showLibrarySeason(command)
	}
}

func (b *Bot) showLibrarySeason(command *userLibrary) bool {
	// Sort series seasons in descending order
	sort.Slice(command.series.Seasons, func(i, j int) bool {
		return command.series.Seasons[i].SeasonNumber > command.series.Seasons[j].SeasonNumber
	})
	series := command.series
	messageText := fmt.Sprintf("[%v](https://www.imdb.com/title/%v) \\- _%v_\n\n", utils.Escape(series.Title), series.ImdbID, series.Year)

	var seasonKeyboardButtons [][]tgbotapi.InlineKeyboardButton
	for _, season := range command.series.Seasons {
		var buttonText string
		if season.SeasonNumber == 0 {
			buttonText = "Specials"
		} else {
			buttonText = fmt.Sprintf("Season %d", season.SeasonNumber)
		}
		row := []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(buttonText, fmt.Sprintf("SEASON_%d", season.SeasonNumber)),
		}
		seasonKeyboardButtons = append(seasonKeyboardButtons, row)
	}

	keyboardMarkup := tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: seasonKeyboardButtons,
	}

	keyboardGoBack := b.createKeyboard(
		[]string{"\U0001F519"},
		[]string{LibrarySeasonEditGoBack},
	)

	keyboardMarkup.InlineKeyboard = append(keyboardMarkup.InlineKeyboard, keyboardGoBack.InlineKeyboard...)

	editMsg := tgbotapi.NewEditMessageTextAndMarkup(
		command.chatID,
		command.messageID,
		messageText,
		keyboardMarkup,
	)
	editMsg.ParseMode = "MarkdownV2"
	editMsg.DisableWebPagePreview = true
	b.setLibraryState(command.chatID, command)
	b.sendMessage(editMsg)
	return false
}

func (b *Bot) handleLibrarySeasonEditToggleMonitor(command *userLibrary) bool {
	command.selectedMonitoring = !command.selectedMonitoring
	b.setLibraryState(command.chatID, command)
	return b.showLibrarySeriesEdit(command)
}

func (b *Bot) handleLibrarySeasonEditSelectSeason(update tgbotapi.Update, command *userLibrary) bool {
	seasonNumberStr := strings.TrimPrefix(update.CallbackQuery.Data, "SEASON_")
	seasonNumber, err := strconv.Atoi(seasonNumberStr)
	if err != nil {
		log.Println("Failed to convert season number to integer:", err)
		return false
	}
	command.selectedSeason = seasonNumber

	b.setLibraryState(command.chatID, command)
	return b.showLibrarySeriesSeasonDetail(command)
}

func (b *Bot) showLibrarySeriesSeasonDetail(command *userLibrary) bool {
	series := command.series
	season := command.seriesSeasons[command.selectedSeason]

	var monitorIcon string
	if season.Monitored {
		monitorIcon = MonitorIcon
	} else {
		monitorIcon = UnmonitorIcon
	}

	var lastSearchString string
	if command.lastSeasonSearch.IsZero() {
		lastSearchString = "" // Set empty string if the time is the zero value
	} else {
		lastSearchString = command.lastSeasonSearch.Format("02 Jan 06 - 15:04") // Convert non-zero time to string
	}

	//get all episodes of a season
	var seasonEpisodes []*sonarr.Episode
	for _, episode := range command.allEpisodes {
		if episode.SeasonNumber == season.SeasonNumber {
			seasonEpisodes = append(seasonEpisodes, episode)

		}
	}

	// iterate over episodes and get their amount and size
	var episodeFilesCounter int
	var totalSize int64
	for _, file := range command.allEpisodeFiles {
		if file.SeasonNumber == season.SeasonNumber {
			episodeFilesCounter++
			totalSize += file.Size
		}
	}

	// Create a message with season details
	var message strings.Builder
	if season.SeasonNumber == 0 {
		fmt.Fprintf(&message, "[%v](https://www.imdb.com/title/%v) \\- _%v_ \\- Specials\n\n", utils.Escape(series.Title), series.ImdbID, series.Year)
	} else {
		fmt.Fprintf(&message, "[%v](https://www.imdb.com/title/%v) \\- _%v_ \\- Season _%v_\n\n", utils.Escape(series.Title), series.ImdbID, series.Year, season.SeasonNumber)
	}
	fmt.Fprintf(&message, "Monitored: %s\n", monitorIcon)
	fmt.Fprintf(&message, "Last Manual Search: %s\n", utils.Escape(lastSearchString))
	fmt.Fprintf(&message, "Episodes: %d\n", len(seasonEpisodes))
	fmt.Fprintf(&message, "Episodes on Disk: %d\n", episodeFilesCounter)
	fmt.Fprintf(&message, "Size: %d GB\n", totalSize/(1024*1024*1024))

	messageText := message.String()

	var keyboard tgbotapi.InlineKeyboardMarkup
	if !season.Monitored {
		keyboard = b.createKeyboard(
			[]string{"Monitor Season", "Monitor Season & Search Now", "Delete Season", "\U0001F519"},
			[]string{LibrarySeasonMonitor, LibrarySeasonMonitorSearchNow, LibrarySeasonDelete, LibrarySeasonGoBack},
		)
	} else {
		keyboard = b.createKeyboard(
			[]string{"Unmonitor Season", "Search Season", "Delete Season", "\U0001F519"},
			[]string{LibrarySeasonUnmonitor, LibrarySeasonSearch, LibrarySeasonDelete, LibrarySeasonGoBack},
		)
	}
	// // Send the message containing series details along with the keyboard
	editMsg := tgbotapi.NewEditMessageTextAndMarkup(
		command.chatID,
		command.messageID,
		messageText,
		keyboard,
	)
	editMsg.ParseMode = "MarkdownV2"
	editMsg.DisableWebPagePreview = true
	b.setLibraryState(command.chatID, command)
	b.sendMessage(editMsg)
	return false
}

func (b *Bot) handleLibrarySeriesSeasonMonitor(command *userLibrary) bool {
	// Access the specific season
	season := getSeasonByNumber(command.series, command.selectedSeason)
	if season == nil {
		log.Println("Invalid season number:", command.selectedSeason)
		return false
	}

	// Update the Monitored field of the season
	season.Monitored = *starr.True()

	// Convert the updated series to AddSeriesInput
	input := seriesToAddSeriesInput(command.series)
	input.Seasons[0].Monitored = *starr.True()

	// Update the series on the server
	_, err := b.SonarrServer.UpdateSeries(input, *starr.False())
	if err != nil {
		msg := tgbotapi.NewMessage(command.chatID, err.Error())
		b.sendMessage(msg)
		return false
	}

	// Monitor al episodes of the season
	var episodeIDs []int64
	for _, episode := range command.seasonEpisodes {
		episodeIDs = append(episodeIDs, episode.ID)
	}

	_, err = b.SonarrServer.MonitorEpisode(episodeIDs, *starr.True())
	if err != nil {
		msg := tgbotapi.NewMessage(command.chatID, err.Error())
		b.sendMessage(msg)
		return false
	}

	b.setLibraryState(command.chatID, command)
	return b.showLibrarySeriesSeasonDetail(command)
}

func (b *Bot) handleLibrarySeriesSeasonUnmonitor(command *userLibrary) bool {
	// Access the specific season
	season := getSeasonByNumber(command.series, command.selectedSeason)
	if season == nil {
		log.Println("Invalid season number:", command.selectedSeason)
		return false
	}

	// Update the Monitored field of the season
	season.Monitored = *starr.False()

	// Convert the updated series to AddSeriesInput
	input := seriesToAddSeriesInput(command.series)

	// Update the series on the server
	_, err := b.SonarrServer.UpdateSeries(input, *starr.False())
	if err != nil {
		msg := tgbotapi.NewMessage(command.chatID, err.Error())
		b.sendMessage(msg)
		return false
	}

	// Unmonitor al episodes of the season
	var episodeIDs []int64
	for _, episode := range command.seasonEpisodes {
		episodeIDs = append(episodeIDs, episode.ID)
	}

	// does not work
	_, err = b.SonarrServer.MonitorEpisode(episodeIDs, *starr.False())
	if err != nil {
		msg := tgbotapi.NewMessage(command.chatID, err.Error())
		b.sendMessage(msg)
		return false
	}

	b.setLibraryState(command.chatID, command)
	return b.showLibrarySeriesSeasonDetail(command)
}

func (b *Bot) handleLibrarySeriesSeasonSearch(command *userLibrary) bool {
	// Monitor al episodes of the season
	var episodeIDs []int64
	for _, episode := range command.seasonEpisodes {
		episodeIDs = append(episodeIDs, episode.ID)
	}
	_, err := b.SonarrServer.MonitorEpisode(episodeIDs, *starr.True())
	if err != nil {
		msg := tgbotapi.NewMessage(command.chatID, err.Error())
		b.sendMessage(msg)
		return false
	}

	cmd := sonarr.CommandRequest{
		Name:         "SeasonSearch",
		SeriesID:     command.series.ID,
		SeasonNumber: command.selectedSeason,
	}
	_, err = b.SonarrServer.SendCommand(&cmd)
	if err != nil {
		msg := tgbotapi.NewMessage(command.chatID, err.Error())
		b.sendMessage(msg)
		return false
	}
	command.lastSeasonSearch = time.Now()
	b.setLibraryState(command.chatID, command)
	return b.showLibrarySeriesSeasonDetail(command)
}

func (b *Bot) handleLibrarySeriesSeasonMonitorSearchNow(command *userLibrary) bool {
	// Access the specific season
	season := getSeasonByNumber(command.series, command.selectedSeason)
	if season == nil {
		log.Println("Invalid season number:", command.selectedSeason)
		return false
	}
	// Update the Monitored field of the season
	season.Monitored = *starr.True()
	input := seriesToAddSeriesInput(command.series)
	_, err := b.SonarrServer.UpdateSeries(input, *starr.False())
	if err != nil {
		msg := tgbotapi.NewMessage(command.chatID, err.Error())
		b.sendMessage(msg)
		return false
	}

	// Monitor al episodes of the season
	var episodeIDs []int64
	for _, episode := range command.seasonEpisodes {
		episodeIDs = append(episodeIDs, episode.ID)
	}
	_, err = b.SonarrServer.MonitorEpisode(episodeIDs, *starr.True())
	if err != nil {
		msg := tgbotapi.NewMessage(command.chatID, err.Error())
		b.sendMessage(msg)
		return false
	}

	cmd := sonarr.CommandRequest{
		Name:         "SeasonSearch",
		SeriesID:     command.series.ID,
		SeasonNumber: command.selectedSeason,
	}
	_, err = b.SonarrServer.SendCommand(&cmd)
	if err != nil {
		msg := tgbotapi.NewMessage(command.chatID, err.Error())
		b.sendMessage(msg)
		return false
	}
	command.lastSeasonSearch = time.Now()
	b.setLibraryState(command.chatID, command)
	return b.showLibrarySeriesSeasonDetail(command)
}

func (b *Bot) handleLibrarySeasonDeleteSeason(command *userLibrary) bool {
	// Access the specific season
	season := getSeasonByNumber(command.series, command.selectedSeason)
	if season == nil {
		log.Println("Invalid season number:", command.selectedSeason)
		return false
	}
	episodes, err := b.SonarrServer.GetSeriesEpisodeFiles(command.series.ID)
	if err != nil {
		msg := tgbotapi.NewMessage(command.chatID, err.Error())
		b.sendMessage(msg)
		return false
	}
	for _, episode := range episodes {
		if episode.SeasonNumber == season.SeasonNumber {
			err := b.SonarrServer.DeleteEpisodeFile(episode.ID)
			if err != nil {
				msg := tgbotapi.NewMessage(command.chatID, err.Error())
				b.sendMessage(msg)
				return false
			}

		}

	}

	series, err := b.SonarrServer.GetSeriesByID(command.series.ID)
	if err != nil {
		msg := tgbotapi.NewMessage(command.chatID, err.Error())
		b.sendMessage(msg)
		return false
	}
	command.series = series

	b.setLibraryState(command.chatID, command)
	return b.showLibrarySeriesSeasonDetail(command)
}

func (b *Bot) handleLibrarySeasonEditSubmitChanges(update tgbotapi.Update, command *userLibrary) bool {
	command.series.Monitored = command.selectedMonitoring
	command.series.QualityProfileID = command.selectedQualityProfile
	command.series.Tags = command.selectedTags
	input := seriesToAddSeriesInput(command.series)
	_, err := b.SonarrServer.UpdateSeries(input, *starr.False())
	if err != nil {
		msg := tgbotapi.NewMessage(command.chatID, err.Error())
		b.sendMessage(msg)
		return false
	}

	text := fmt.Sprintf("Series '%v' updated\n", command.series.Title)
	b.clearState(update)
	b.sendMessageWithEdit(command, text)
	return true
}
