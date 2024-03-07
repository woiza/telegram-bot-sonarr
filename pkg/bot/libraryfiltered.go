package bot

import (
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/woiza/telegram-bot-sonarr/pkg/utils"
	"golift.io/starr"
	"golift.io/starr/sonarr"
)

const (
	LibrarySeriesDelete     = "LIBRARY_SERIES_DELETE"
	LibrarySeriesDeleteYes  = "LIBRARY_SERIES_DELETE_YES"
	LibrarySeriesDeleteNo   = "LIBRARY_SERIES_DELETE_NO"
	LibrarySeriesEdit       = "LIBRARY_SERIES_EDIT"
	LibrarySeriesSeasonEdit = "LIBRARY_SERIES_SEASON_EDIT"
	LibrarySeriesGoBack     = "LIBRARY_SERIES_GOBACK"
	//LibraryFilteredGoBack        = "LIBRARY_FILTERED_GOBACK" already defined in librarymenu.go
	LibrarySeriesMonitor          = "LIBRARY_SERIES_MONITOR"
	LibrarySeriesUnmonitor        = "LIBRARY_SERIES_UNMONITOR"
	LibrarySeriesSearch           = "LIBRARY_SERIES_SEARCH"
	LibrarySeriesMonitorSearchNow = "LIBRARY_SERIES_MONITOR_SEARCHNOW"
	LibraryFilteredActive         = "LIBRARYFILTERED"
	//LibraryMenuActive            = "LIBRARYMENU" already defined in librarymenu.go

)

const (
	MonitorIcon   = "\u2705" // Green checkmark
	UnmonitorIcon = "\u274C" // Red X
)

func (b *Bot) libraryFiltered(update tgbotapi.Update) bool {
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
	// ignore click on page number
	case "current_page":
		return false
	case LibraryFirstPage:
		command.page = 0
		return b.showLibraryMenuFiltered(command)
	case LibraryPreviousPage:
		if command.page > 0 {
			command.page--
		}
		return b.showLibraryMenuFiltered(command)
	case LibraryNextPage:
		command.page++
		return b.showLibraryMenuFiltered(command)
	case LibraryLastPage:
		totalPages := (len(command.libraryFiltered) + b.Config.MaxItems - 1) / b.Config.MaxItems
		command.page = totalPages - 1
		return b.showLibraryMenuFiltered(command)
	case LibrarySeriesGoBack:
		command.series = nil
		b.setActiveCommand(userID, LibraryFilteredActive)
		b.setLibraryState(command.chatID, command)
		return b.showLibraryMenuFiltered(command)
	case LibraryFilteredGoBack:
		command.filter = ""
		b.setActiveCommand(userID, LibraryMenuActive)
		b.setLibraryState(command.chatID, command)
		return b.showLibraryMenu(command)
	case LibrarySeriesMonitor:
		return b.handleLibrarySeriesMonitor(update, command)
	case LibrarySeriesUnmonitor:
		return b.handleLibrarySeriesUnMonitor(update, command)
	case LibrarySeriesSearch:
		return b.handleLibrarySeriesSearch(update, command)
	case LibrarySeriesDelete:
		return b.handleLibrarySeriesDelete(command)
	case LibrarySeriesDeleteYes:
		return b.handleLibrarySeriesDeleteYes(update, command)
	case LibrarySeriesDeleteNo:
		return b.showLibrarySeriesDetail(update, command)
	case LibrarySeriesEdit:
		return b.handleLibrarySeriesEdit(command)
	case LibrarySeriesSeasonEdit:
		return b.handleLibrarySeasonsEdit(command)
	case LibrarySeriesMonitorSearchNow:
		return b.handleLibrarySeriesMonitorSearchNow(update, command)
	default:
		return b.showLibrarySeriesDetail(update, command)
	}
}

func (b *Bot) showLibrarySeriesDetail(update tgbotapi.Update, command *userLibrary) bool {
	var series *sonarr.Series
	if command.series == nil {
		seriesIDStr := strings.TrimPrefix(update.CallbackQuery.Data, "TVDBID_")
		series = command.libraryFiltered[seriesIDStr]
		command.series = series

	} else {
		series = command.series
	}

	command.selectedMonitoring = series.Monitored
	command.selectedTags = series.Tags
	command.selectedQualityProfile = series.QualityProfileID

	var monitorIcon string
	if series.Monitored {
		monitorIcon = MonitorIcon
	} else {
		monitorIcon = UnmonitorIcon
	}

	var lastSearchString string
	if command.lastSearch.IsZero() {
		lastSearchString = "" // Set empty string if the time is the zero value
	} else {
		lastSearchString = command.lastSearch.Format("02 Jan 06 - 15:04") // Convert non-zero time to string
	}

	var tagLabels []string
	for _, tagID := range series.Tags {
		tag := findTagByID(command.allTags, tagID)
		tagLabels = append(tagLabels, tag.Label)
		command.selectedTags = append(command.selectedTags, tag.ID)
	}
	tagsString := strings.Join(tagLabels, ", ")

	// seriesFiles, err := b.SonarrServer.GetSeriesEpisodeFiles(series.ID)
	// if err != nil {
	// 	msg := tgbotapi.NewMessage(command.chatID, err.Error())
	// 	b.sendMessage(msg)
	// 	return false
	// }

	// Create a message with series details
	var message strings.Builder
	fmt.Fprintf(&message, "[%v](https://www.imdb.com/title/%v) \\- _%v_\n\n", utils.Escape(series.Title), series.ImdbID, series.Year)
	fmt.Fprintf(&message, "Monitored: %s\n", monitorIcon)
	fmt.Fprintf(&message, "Status: %s\n", utils.Escape(series.Status))
	fmt.Fprintf(&message, "Last Manual Search: %s\n", utils.Escape(lastSearchString))
	fmt.Fprintf(&message, "Size: %d GB\n", series.Statistics.SizeOnDisk/(1024*1024*1024))
	fmt.Fprintf(&message, "Tags: %s\n", utils.Escape(tagsString))
	fmt.Fprintf(&message, "Quality Profile: %s\n", utils.Escape(getQualityProfileByID(command.qualityProfiles, series.QualityProfileID).Name))
	//fmt.Fprintf(&message, "Custom Format Score: %s\n", utils.Escape(customFormatScore))

	messageText := message.String()

	var keyboard tgbotapi.InlineKeyboardMarkup
	if !series.Monitored {
		keyboard = b.createKeyboard(
			[]string{"Monitor Series", "Monitor Series & Search Now", "Delete Series", "Edit Series", "Edit Seasons", "\U0001F519"},
			[]string{LibrarySeriesMonitor, LibrarySeriesMonitorSearchNow, LibrarySeriesDelete, LibrarySeriesEdit, LibrarySeriesSeasonEdit, LibrarySeriesGoBack},
		)
	} else {
		keyboard = b.createKeyboard(
			[]string{"Unmonitor Series", "Search Series", "Delete Series", "Edit Series", "Edit Seasons", "\U0001F519"},
			[]string{LibrarySeriesUnmonitor, LibrarySeriesSearch, LibrarySeriesDelete, LibrarySeriesEdit, LibrarySeriesSeasonEdit, LibrarySeriesGoBack},
		)
	}

	// Send the message containing series details along with the keyboard
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

func (b *Bot) handleLibrarySeriesMonitor(update tgbotapi.Update, command *userLibrary) bool {
	command.series.Monitored = *starr.True()
	input := seriesToAddSeriesInput(command.series)
	_, err := b.SonarrServer.UpdateSeries(input, *starr.False())
	if err != nil {
		msg := tgbotapi.NewMessage(command.chatID, err.Error())
		b.sendMessage(msg)
		return false
	}
	b.setLibraryState(command.chatID, command)
	return b.showLibrarySeriesDetail(update, command)
}

func (b *Bot) handleLibrarySeriesUnMonitor(update tgbotapi.Update, command *userLibrary) bool {
	command.series.Monitored = *starr.False()
	input := seriesToAddSeriesInput(command.series)
	_, err := b.SonarrServer.UpdateSeries(input, *starr.False())
	if err != nil {
		msg := tgbotapi.NewMessage(command.chatID, err.Error())
		b.sendMessage(msg)
		return false
	}
	b.setLibraryState(command.chatID, command)
	return b.showLibrarySeriesDetail(update, command)
}

func (b *Bot) handleLibrarySeriesSearch(update tgbotapi.Update, command *userLibrary) bool {
	cmd := sonarr.CommandRequest{
		Name:     "SeriesSearch",
		SeriesID: command.series.ID,
	}
	_, err := b.SonarrServer.SendCommand(&cmd)
	if err != nil {
		msg := tgbotapi.NewMessage(command.chatID, err.Error())
		b.sendMessage(msg)
		return false
	}
	command.lastSearch = time.Now()
	b.setLibraryState(command.chatID, command)
	return b.showLibrarySeriesDetail(update, command)
}

func (b *Bot) handleLibrarySeriesMonitorSearchNow(update tgbotapi.Update, command *userLibrary) bool {
	command.series.Monitored = *starr.True()
	input := seriesToAddSeriesInput(command.series)
	_, err := b.SonarrServer.UpdateSeries(input, *starr.False())
	if err != nil {
		msg := tgbotapi.NewMessage(command.chatID, err.Error())
		b.sendMessage(msg)
		return false
	}

	cmd := sonarr.CommandRequest{
		Name:     "SeriesSearch",
		SeriesID: command.series.ID,
	}
	_, err = b.SonarrServer.SendCommand(&cmd)
	if err != nil {
		msg := tgbotapi.NewMessage(command.chatID, err.Error())
		b.sendMessage(msg)
		return false
	}
	command.lastSearch = time.Now()
	b.setLibraryState(command.chatID, command)
	return b.showLibrarySeriesDetail(update, command)
}

func (b *Bot) handleLibrarySeriesDelete(command *userLibrary) bool {
	messageText := fmt.Sprintf("[%v](https://www.imdb.com/title/%v) \\- _%v_\n\n", utils.Escape(command.series.Title), command.series.ImdbID, command.series.Year)
	keyboard := b.createKeyboard(
		[]string{"Yes, delete this series", "\U0001F519"},
		[]string{LibrarySeriesDeleteYes, LibrarySeriesDeleteNo},
	)
	// Send the message containing series details along with the keyboard
	editMsg := tgbotapi.NewEditMessageTextAndMarkup(
		command.chatID,
		command.messageID,
		messageText,
		keyboard,
	)
	editMsg.ParseMode = "MarkdownV2"
	editMsg.DisableWebPagePreview = false
	b.setLibraryState(command.chatID, command)
	b.sendMessage(editMsg)
	return false
}

func (b *Bot) handleLibrarySeriesDeleteYes(update tgbotapi.Update, command *userLibrary) bool {
	err := b.SonarrServer.DeleteSeries(int(command.series.ID), *starr.True(), *starr.False())
	if err != nil {
		msg := tgbotapi.NewMessage(command.chatID, err.Error())
		b.sendMessage(msg)
		return false
	}
	text := fmt.Sprintf("Series '%v' deleted\n", command.series.Title)
	b.clearState(update)
	b.sendMessageWithEdit(command, text)
	return true
}

func (b *Bot) handleLibrarySeriesEdit(command *userLibrary) bool {
	b.setLibraryState(command.chatID, command)
	b.setActiveCommand(command.chatID, LibrarySeriesEditCommand)
	return b.showLibrarySeriesEdit(command)
}

func (b *Bot) handleLibrarySeasonsEdit(command *userLibrary) bool {
	b.setLibraryState(command.chatID, command)
	b.setActiveCommand(command.chatID, LibrarySeasonsEditCommand)
	return b.showLibrarySeasons(command)
}
