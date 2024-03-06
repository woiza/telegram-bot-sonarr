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
	DeleteSeriesConfirm      = "DELETE_SERIES_SUBMIT"
	DeleteSeriesCancel       = "DELETE_SERIES_CANCEL"
	DeleteSeriesGoBack       = "DELETE_SERIES_GOBACK"
	DeleteSeriesYes          = "DELETE_SERIES_YES"
	DeleteSeriesTvdbID       = "DELETE_SERIES_TVDBID_"
	DeleteSeriesFirstPage    = "DELETE_SERIES_FIRST_PAGE"
	DeleteSeriesPreviousPage = "DELETE_SERIES_PREV_PAGE"
	DeleteSeriesNextPage     = "DELETE_SERIES_NEXT_PAGE"
	DeleteSeriesLastPage     = "DELETE_SERIES_LAST_PAGE"
)

func (b *Bot) processDeleteCommand(update tgbotapi.Update, userID int64, s *sonarr.Sonarr) {
	msg := tgbotapi.NewMessage(userID, "Handling delete command... please wait")
	message, _ := b.sendMessage(msg)

	series, err := s.GetSeries(0)
	if err != nil {
		msg := tgbotapi.NewMessage(userID, err.Error())
		b.sendMessage(msg)
		return
	}
	command := userDeleteSeries{
		library: make(map[string]*sonarr.Series, len(series)),
	}
	for _, series := range series {
		tvdbid := strconv.Itoa(int(series.TvdbID))
		command.library[tvdbid] = series
	}

	// Sort the Series alphabetically based on their titles
	sort.SliceStable(series, func(i, j int) bool {
		return utils.IgnoreArticles(strings.ToLower(series[i].Title)) < utils.IgnoreArticles(strings.ToLower(series[j].Title))
	})
	command.seriesForSelection = series
	command.chatID = message.Chat.ID
	command.messageID = message.MessageID
	b.setDeleteSeriesState(userID, &command)

	criteria := update.Message.CommandArguments()
	// no search criteria --> show complete library and return
	if len(criteria) < 1 {
		b.showDeleteSerieSelection(&command)
		return
	}

	searchResults, err := s.Lookup(criteria)
	if err != nil {
		msg := tgbotapi.NewMessage(userID, err.Error())
		b.sendMessage(msg)
		return
	}

	b.setDeleteSeriesState(userID, &command)
	b.handleDeleteSearchResults(searchResults, &command)

}
func (b *Bot) deleteSeries(update tgbotapi.Update) bool {
	userID, err := b.getUserID(update)
	if err != nil {
		fmt.Printf("Cannot delete Series: %v", err)
		return false
	}

	command, exists := b.getDeleteSeriesState(userID)
	if !exists {
		return false
	}

	switch update.CallbackQuery.Data {
	// ignore click on page number
	case "current_page":
		return false
	case DeleteSeriesFirstPage:
		command.page = 0
		return b.showDeleteSerieSelection(command)
	case DeleteSeriesPreviousPage:
		if command.page > 0 {
			command.page--
		}
		return b.showDeleteSerieSelection(command)
	case DeleteSeriesNextPage:
		command.page++
		return b.showDeleteSerieSelection(command)
	case DeleteSeriesLastPage:
		totalPages := (len(command.seriesForSelection) + b.Config.MaxItems - 1) / b.Config.MaxItems
		command.page = totalPages - 1
		return b.showDeleteSerieSelection(command)
	case DeleteSeriesConfirm:
		return b.processSerieSelectionForDelete(command)
	case DeleteSeriesYes:
		return b.handleDeleteSeriesYes(update, command)
	case DeleteSeriesGoBack:
		return b.showDeleteSerieSelection(command)
	case DeleteSeriesCancel:
		b.clearState(update)
		b.sendMessageWithEdit(command, CommandsCleared)
		return false
	default:
		// Check if it starts with DELETESeries_TVDBID_
		if strings.HasPrefix(update.CallbackQuery.Data, DeleteSeriesTvdbID) {
			return b.handleDeleteSerieSelection(update, command)
		}
		return false
	}
}

func (b *Bot) showDeleteSerieSelection(command *userDeleteSeries) bool {
	var keyboard tgbotapi.InlineKeyboardMarkup

	series := command.seriesForSelection

	// Pagination parameters
	page := command.page
	pageSize := b.Config.MaxItems
	totalPages := (len(series) + pageSize - 1) / pageSize

	// Calculate start and end index for the current page
	startIndex := page * pageSize
	endIndex := (page + 1) * pageSize
	if endIndex > len(series) {
		endIndex = len(series)
	}

	var seriesKeyboard [][]tgbotapi.InlineKeyboardButton
	for _, series := range series[startIndex:endIndex] {
		// Check if the Series is selected
		isSelected := isSelectedSeries(command.selectedSeries, series.ID)

		// Create button text with or without check mark
		buttonText := series.Title
		if isSelected {
			buttonText += " \u2705"
		}

		row := []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(buttonText, DeleteSeriesTvdbID+strconv.Itoa(int(series.TvdbID))),
		}
		seriesKeyboard = append(seriesKeyboard, row)
	}

	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, seriesKeyboard...)

	// Create pagination buttons
	if len(series) > pageSize {
		paginationButtons := []tgbotapi.InlineKeyboardButton{}
		if page > 0 {
			paginationButtons = append(paginationButtons, tgbotapi.NewInlineKeyboardButtonData("◀️", DeleteSeriesPreviousPage))
		}
		paginationButtons = append(paginationButtons, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%d/%d", page+1, totalPages), "current_page"))
		if page+1 < totalPages {
			paginationButtons = append(paginationButtons, tgbotapi.NewInlineKeyboardButtonData("▶️", DeleteSeriesNextPage))
		}
		if page != 0 {
			paginationButtons = append([]tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData("⏮️", DeleteSeriesFirstPage)}, paginationButtons...)
		}
		if page+1 != totalPages {
			paginationButtons = append(paginationButtons, tgbotapi.NewInlineKeyboardButtonData("⏭️", DeleteSeriesLastPage))
		}

		keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, paginationButtons)
	}

	var keyboardConfirmCancel tgbotapi.InlineKeyboardMarkup
	if len(command.selectedSeries) > 0 {
		keyboardConfirmCancel = b.createKeyboard(
			[]string{"Submit - Confirm Series", "Cancel - clear command"},
			[]string{DeleteSeriesConfirm, DeleteSeriesCancel},
		)
	} else {
		keyboardConfirmCancel = b.createKeyboard(
			[]string{"Cancel - clear command"},
			[]string{DeleteSeriesCancel},
		)
	}

	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, keyboardConfirmCancel.InlineKeyboard...)

	// Send the message containing Series details along with the keyboard
	editMsg := tgbotapi.NewEditMessageTextAndMarkup(
		command.chatID,
		command.messageID,
		fmt.Sprintf(utils.Escape("Select the Series(s) you want to delete - page %d/%d"), page+1, totalPages),
		keyboard,
	)
	editMsg.ParseMode = "MarkdownV2"
	editMsg.DisableWebPagePreview = true
	b.setDeleteSeriesState(command.chatID, command)
	b.sendMessage(editMsg)
	return false
}

func (b *Bot) handleDeleteSearchResults(searchResults []*sonarr.Series, command *userDeleteSeries) {
	if len(searchResults) == 0 {
		b.sendMessageWithEdit(command, "No Series found matching your search criteria")
		return
	}
	if len(searchResults) > 25 {
		b.sendMessageWithEdit(command, "Result size too large, please narrow down your search criteria")
		return
	}

	// if Series has a radarr ID, it's in the library
	var SeriesInLibrary []*sonarr.Series
	for _, Series := range searchResults {
		if Series.ID != 0 {
			SeriesInLibrary = append(SeriesInLibrary, Series)
		}
	}
	if len(SeriesInLibrary) == 0 {
		b.sendMessageWithEdit(command, "No Series found in your library")
		return
	}

	if len(SeriesInLibrary) == 1 {
		command.selectedSeries = make([]*sonarr.Series, len(SeriesInLibrary))
		command.selectedSeries[0] = SeriesInLibrary[0]
		b.setDeleteSeriesState(command.chatID, command)
		b.processSerieSelectionForDelete(command)
	} else {
		command.seriesForSelection = SeriesInLibrary
		b.setDeleteSeriesState(command.chatID, command)
		b.showDeleteSerieSelection(command)
	}
}
func (b *Bot) processSerieSelectionForDelete(command *userDeleteSeries) bool {
	var keyboard tgbotapi.InlineKeyboardMarkup
	var messageText strings.Builder
	var disablePreview bool
	switch len(command.selectedSeries) {
	case 1:
		keyboard = b.createKeyboard(
			[]string{"Yes, delete this Series", "Cancel, clear command", "\U0001F519"},
			[]string{DeleteSeriesYes, DeleteSeriesCancel, DeleteSeriesGoBack},
		)
		fmt.Fprintf(&messageText, "Do you want to delete the following series including all files?\n\n")
		fmt.Fprintf(&messageText, "[%v](https://www.imdb.com/title/%v) \\- _%v_\n",
			utils.Escape(command.selectedSeries[0].Title), command.selectedSeries[0].ImdbID, command.selectedSeries[0].Year)
		disablePreview = false
	case 0:
		return b.showDeleteSerieSelection(command)
	default:
		keyboard = b.createKeyboard(
			[]string{"Yes, delete these Series", "Cancel, clear command", "\U0001F519"},
			[]string{DeleteSeriesYes, DeleteSeriesCancel, DeleteSeriesGoBack},
		)

		fmt.Fprintf(&messageText, "Do you want to delete the following series including all files?\n\n")
		for _, Series := range command.selectedSeries {
			fmt.Fprintf(&messageText, "[%v](https://www.imdb.com/title/%v) \\- _%v_\n",
				utils.Escape(Series.Title), Series.ImdbID, Series.Year)
		}
		disablePreview = true
	}

	editMsg := tgbotapi.NewEditMessageTextAndMarkup(
		command.chatID,
		command.messageID,
		messageText.String(),
		keyboard,
	)

	editMsg.ParseMode = "MarkdownV2"
	editMsg.DisableWebPagePreview = disablePreview
	editMsg.ReplyMarkup = &keyboard

	b.setDeleteSeriesState(command.chatID, command)
	b.sendMessage(editMsg)
	return false
}

func (b *Bot) handleDeleteSeriesYes(update tgbotapi.Update, command *userDeleteSeries) bool {
	for _, series := range command.selectedSeries {
		err := b.SonarrServer.DeleteSeries(int(series.ID), *starr.True(), *starr.False())
		if err != nil {
			msg := tgbotapi.NewMessage(command.chatID, err.Error())
			fmt.Println(err)
			b.sendMessage(msg)
			return false
		}
	}

	deletedSeries := make([]string, len(command.selectedSeries))
	for i, series := range command.selectedSeries {
		deletedSeries[i] = series.Title
	}

	messageText := fmt.Sprintf("Deleted Series:\n- %v", strings.Join(deletedSeries, "\n- "))
	editMsg := tgbotapi.NewEditMessageText(
		command.chatID,
		command.messageID,
		messageText,
	)

	b.clearState(update)
	b.sendMessage(editMsg)
	return true
}

func (b *Bot) handleDeleteSerieSelection(update tgbotapi.Update, command *userDeleteSeries) bool {
	seriesIDStr := strings.TrimPrefix(update.CallbackQuery.Data, DeleteSeriesTvdbID)
	series := command.library[seriesIDStr]

	// Check if the Series is already selected
	if isSelectedSeries(command.selectedSeries, series.ID) {
		// If selected, remove the Series from selectedSeries (deselect)
		command.selectedSeries = removeSeries(command.selectedSeries, series.ID)
	} else {
		// If not selected, add the Series to selectedSeries (select)
		command.selectedSeries = append(command.selectedSeries, series)
	}
	b.setDeleteSeriesState(command.chatID, command)

	return b.showDeleteSerieSelection(command)
}

func isSelectedSeries(selectedSeries []*sonarr.Series, SeriesID int64) bool {
	for _, selectedSeries := range selectedSeries {
		if selectedSeries.ID == SeriesID {
			return true
		}
	}
	return false
}

func removeSeries(selectedSeries []*sonarr.Series, SeriesID int64) []*sonarr.Series {
	var updatedSeries []*sonarr.Series
	for _, series := range selectedSeries {
		if series.ID != SeriesID {
			updatedSeries = append(updatedSeries, series)
		}
	}
	return updatedSeries
}
