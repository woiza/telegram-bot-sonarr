package bot

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/woiza/telegram-bot-sonarr/pkg/utils"
	"golift.io/starr"
	"golift.io/starr/sonarr"
)

const (
	LibraryFilteredGoBack = "LIBRARY_FILTERED_GOBACK"
	LibraryMenu           = "LIBRARY_MENU"
	LibraryCancel         = "LIBRARY_CANCEL"
	LibraryMenuActive     = "LIBRARYMENU"
	LibraryFiltered       = "LIBRARYFILTERED"
	LibraryFirstPage      = "LIBRARY_FIRST_PAGE"
	LibraryPreviousPage   = "LIBRARY_PREV_PAGE"
	LibraryNextPage       = "LIBRARY_NEXT_PAGE"
	LibraryLastPage       = "LIBRARY_LAST_PAGE"
)

const (
	FilterMonitored       = "FILTER_MONITORED"
	FilterUnmonitored     = "FILTER_UNMONITORED"
	FilterContinuing      = "FILTER_CONTINUING"
	FilterEnded           = "FILTER_ENDED"
	FilterMissingEpisodes = "FILTER_MISSINGEPISODES"
	FilterOnDisk          = "FILTER_ONDISK"
	FilterShowAll         = "FILTER_SHOWALL"
	FilterSearchResults   = "FILTER_SEARCHRESULTS"
)

func (b *Bot) processLibraryCommand(update tgbotapi.Update, userID int64, s *sonarr.Sonarr) {
	msg := tgbotapi.NewMessage(userID, "Handling library command... please wait")
	message, _ := b.sendMessage(msg)

	qualityProfiles, err := s.GetQualityProfiles()
	if err != nil {
		msg := tgbotapi.NewMessage(userID, err.Error())
		b.sendMessage(msg)
		return
	}
	tags, err := s.GetTags()
	if err != nil {
		msg := tgbotapi.NewMessage(userID, err.Error())
		b.sendMessage(msg)
		return
	}
	series, err := s.GetSeries(0)
	if err != nil {
		msg := tgbotapi.NewMessage(userID, err.Error())
		b.sendMessage(msg)
		return
	}

	command := userLibrary{}
	command.qualityProfiles = qualityProfiles
	command.allTags = tags
	command.library = series
	command.filter = ""
	command.chatID = message.Chat.ID
	command.messageID = message.MessageID

	criteria := update.Message.CommandArguments()
	// no search criteria --> show menu and return
	if len(criteria) < 1 {
		b.setLibraryState(userID, &command)
		b.showLibraryMenu(&command)
		return
	}

	searchResults, err := s.Lookup(criteria)
	if err != nil {
		msg := tgbotapi.NewMessage(userID, err.Error())
		b.sendMessage(msg)
		return
	}

	b.handleSearchResults(update, searchResults, &command)

}

func (b *Bot) libraryMenu(update tgbotapi.Update) bool {
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
	case LibraryFilteredGoBack:
		command.filter = ""
		b.setActiveCommand(userID, LibraryMenuActive)
		b.setLibraryState(command.chatID, command)
		return b.showLibraryMenu(command)
	case LibraryMenu:
		command.filter = ""
		b.setLibraryState(command.chatID, command)
		b.showLibraryMenu(command)
		return false
	case LibraryCancel:
		b.clearState(update)
		b.sendMessageWithEdit(command, CommandsCleared)
		return false
	default:
		command.filter = update.CallbackQuery.Data
		b.setLibraryState(command.chatID, command)
		return b.showLibraryMenuFiltered(command)
	}
}
func (b *Bot) showLibraryMenu(command *userLibrary) bool {
	keyboard := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("Monitored Series", FilterMonitored),
			tgbotapi.NewInlineKeyboardButtonData("Unonitored Series", FilterUnmonitored),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("Continuing Series", FilterContinuing),
			tgbotapi.NewInlineKeyboardButtonData("Ended Series", FilterEnded),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("Series on Disk", FilterOnDisk),
			tgbotapi.NewInlineKeyboardButtonData("Series with missing episodes", FilterMissingEpisodes),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("All Series", FilterShowAll),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("Cancel - clear command", LibraryCancel),
		},
	}
	command.page = 0
	b.setLibraryState(command.chatID, command)
	b.sendMessageWithEditAndKeyboard(command, tgbotapi.InlineKeyboardMarkup{InlineKeyboard: keyboard}, "Select an option:")
	return false
}

func (b *Bot) showLibraryMenuFiltered(command *userLibrary) bool {

	var filteredSeries []*sonarr.Series
	var responseText string

	switch command.filter {
	case FilterMonitored:
		filteredSeries = filterSeries(command.library, func(series *sonarr.Series) bool {
			return series.Monitored
		})
		command.filter = FilterMonitored
		responseText = "Monitored Series"
	case FilterUnmonitored:
		filteredSeries = filterSeries(command.library, func(series *sonarr.Series) bool {
			return !series.Monitored
		})
		command.filter = FilterUnmonitored
		responseText = "Unmonitored Series"
	case FilterContinuing:
		filteredSeries = filterSeries(command.library, func(series *sonarr.Series) bool {
			return series.Status == "continuing"
		})
		command.filter = FilterContinuing
		responseText = "Continuing Series"
	case FilterEnded:
		filteredSeries = filterSeries(command.library, func(series *sonarr.Series) bool {
			return series.Ended == *starr.True()
		})
		command.filter = FilterEnded
		responseText = "Ended Series"
	case FilterOnDisk:
		filteredSeries = filterSeries(command.library, func(series *sonarr.Series) bool {
			for _, season := range series.Seasons {
				if season.Statistics.SizeOnDisk > 0 {
					return true
				}
			}
			return false
		})
		responseText = "Series on Disk"
	case FilterMissingEpisodes:
		filteredSeries = filterSeries(command.library, func(series *sonarr.Series) bool {
			for _, season := range series.Seasons {
				if season.Statistics.EpisodeFileCount < season.Statistics.TotalEpisodeCount {
					return true
				}
			}
			return false
		})
		command.filter = FilterMissingEpisodes
		responseText = "Series with Missing Episodes"
	case FilterShowAll:
		filteredSeries = filterSeries(command.library, func(series *sonarr.Series) bool {
			return true
		})
		command.filter = FilterShowAll
		responseText = "All Series"
	case FilterSearchResults:
		filteredSeries = command.searchResultsInLibrary
		command.filter = FilterSearchResults
		responseText = "Search Results"
	default:
		command.filter = ""
		b.setLibraryState(command.chatID, command)
		return false
	}

	var inlineKeyboard [][]tgbotapi.InlineKeyboardButton
	var row []tgbotapi.InlineKeyboardButton

	if len(filteredSeries) == 0 {
		responseText = "No series found matching your filter criteria"
		row = append(row, tgbotapi.NewInlineKeyboardButtonData("\U0001F519", LibraryFilteredGoBack))
		inlineKeyboard = append(inlineKeyboard, row)
	} else {

		// Pagination parameters
		page := command.page
		pageSize := b.Config.MaxItems
		totalPages := (len(filteredSeries) + pageSize - 1) / pageSize

		// Calculate start and end index for the current page
		startIndex := page * pageSize
		endIndex := (page + 1) * pageSize
		if endIndex > len(filteredSeries) {
			endIndex = len(filteredSeries)
		}

		responseText = fmt.Sprintf("%s - page %d/%d", responseText, page+1, totalPages)

		sort.SliceStable(filteredSeries, func(i, j int) bool {
			return utils.IgnoreArticles(strings.ToLower(filteredSeries[i].Title)) < utils.IgnoreArticles(strings.ToLower(filteredSeries[j].Title))
		})
		inlineKeyboard = b.getSeriesAsInlineKeyboard(filteredSeries[startIndex:endIndex])

		// Create pagination buttons
		if len(filteredSeries) > pageSize {
			paginationButtons := []tgbotapi.InlineKeyboardButton{}
			if page > 0 {
				paginationButtons = append(paginationButtons, tgbotapi.NewInlineKeyboardButtonData("◀️", LibraryPreviousPage))
			}
			paginationButtons = append(paginationButtons, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%d/%d", page+1, totalPages), "current_page"))
			if page+1 < totalPages {
				paginationButtons = append(paginationButtons, tgbotapi.NewInlineKeyboardButtonData("▶️", LibraryNextPage))
			}
			if page != 0 {
				paginationButtons = append([]tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData("⏮️", LibraryFirstPage)}, paginationButtons...)
			}
			if page+1 != totalPages {
				paginationButtons = append(paginationButtons, tgbotapi.NewInlineKeyboardButtonData("⏭️", LibraryLastPage))
			}

			inlineKeyboard = append(inlineKeyboard, paginationButtons)
		}

		row = append(row, tgbotapi.NewInlineKeyboardButtonData("\U0001F519", LibraryFilteredGoBack))
		inlineKeyboard = append(inlineKeyboard, row)
	}

	editMsg := tgbotapi.NewEditMessageTextAndMarkup(
		command.chatID,
		command.messageID,
		responseText,
		tgbotapi.InlineKeyboardMarkup{
			InlineKeyboard: inlineKeyboard,
		},
	)

	command.libraryFiltered = make(map[string]*sonarr.Series, len(filteredSeries))
	for _, series := range filteredSeries {
		tvdbID := strconv.Itoa(int(series.TvdbID))
		command.libraryFiltered[tvdbID] = series
	}

	b.setLibraryState(command.chatID, command)
	b.setActiveCommand(command.chatID, LibraryFiltered)
	b.sendMessage(editMsg)
	return false
}

func filterSeries(series []*sonarr.Series, filterCondition func(movie *sonarr.Series) bool) []*sonarr.Series {
	var filtered []*sonarr.Series
	for _, s := range series {
		if filterCondition(s) {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

func (b *Bot) handleSearchResults(update tgbotapi.Update, searchResults []*sonarr.Series, command *userLibrary) {
	if len(searchResults) == 0 {
		b.sendMessageWithEdit(command, "No series found matching your search criteria")
		return
	}
	if len(searchResults) > 25 {
		b.sendMessageWithEdit(command, "Result size too large, please narrow down your search criteria")
		return
	}

	// if series has a sonarr ID, it's in the library
	var seriesInLibrary []*sonarr.Series
	for _, series := range searchResults {
		if series.ID != 0 {
			seriesInLibrary = append(seriesInLibrary, series)
		}
	}
	if len(seriesInLibrary) == 0 {
		b.sendMessageWithEdit(command, "No series found in your library")
		return
	}

	command.searchResultsInLibrary = seriesInLibrary

	// go to series details
	if len(seriesInLibrary) == 1 {
		command.series = seriesInLibrary[0]
		command.filter = FilterSearchResults
		b.setLibraryState(command.chatID, command)
		b.setActiveCommand(command.chatID, LibraryFilteredCommand)
		b.showLibrarySeriesDetail(update, command)
	} else {
		command.filter = FilterSearchResults
		b.setLibraryState(command.chatID, command)
		b.setActiveCommand(command.chatID, LibraryFilteredCommand)
		b.showLibraryMenuFiltered(command)
	}
}
