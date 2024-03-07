package bot

import (
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/woiza/telegram-bot-sonarr/pkg/utils"
	"golift.io/starr"
)

const (
	LibrarySeasonsEditToggleMonitor = "LIBRARY_SEASONS_EDIT_TOGGLE_MONITOR"
	LibrarySeasonsEditSubmitChanges = "LIBRARY_SEASONS_EDIT_SUBMIT_CHANGES"
	LibrarySeasonsEditGoBack        = "LIBRARY_SEASONS_EDIT_GOBACK"
	LibrarySeasonsEditCancel        = "LIBRARY_SEASONS_EDIT_CANCEL"
)

func (b *Bot) librarySeasonsEdit(update tgbotapi.Update) bool {
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
	case LibrarySeasonsEditToggleMonitor:
		return b.handleLibrarySeasonsEditToggleMonitor(command)
	case LibrarySeasonsEditSubmitChanges:
		return b.handleLibrarySeasonsEditSubmitChanges(update, command)
	case LibrarySeasonsEditGoBack:
		b.setActiveCommand(userID, LibraryFilteredActive)
		b.setLibraryState(command.chatID, command)
		return b.showLibrarySeriesDetail(update, command)
	case LibrarySeasonsEditCancel:
		b.clearState(update)
		b.sendMessageWithEdit(command, CommandsCleared)
		return false
	default:
		// Check if it starts with "TAG_"
		if strings.HasPrefix(update.CallbackQuery.Data, "SEASON_") {
			return b.handleLibrarySeasonsEditSelectTag(update, command)
		}
		return b.showLibrarySeasons(command)
	}
}

func (b *Bot) showLibrarySeasons(command *userLibrary) bool {

	series := command.series

	var seasonsKeyboard [][]tgbotapi.InlineKeyboardButton
	for _, season := range command.series.Seasons {

		row := []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(strconv.Itoa(season.SeasonNumber), "SEASON_"+strconv.Itoa(int(season.SeasonNumber))),
		}
		seasonsKeyboard = append(seasonsKeyboard, row)
	}

	// var monitorIcon string
	// if command.selectedMonitoring {
	// 	monitorIcon = MonitorIcon
	// } else {
	// 	monitorIcon = UnmonitorIcon
	// }

	messageText := fmt.Sprintf("[%v](https://www.imdb.com/title/%v) \\- _%v_\n\n", utils.Escape(series.Title), series.ImdbID, series.Year)

	keyboard := b.createKeyboard(
		[]string{"Monitored: "},
		[]string{LibrarySeasonsEditToggleMonitor},
	)

	// var tagsKeyboard [][]tgbotapi.InlineKeyboardButton
	// for _, tag := range command.allTags {
	// 	// Check if the tag is selected
	// 	isSelected := isSelectedTag(command.selectedTags, tag.ID)

	// 	var buttonText string
	// 	if isSelected {
	// 		buttonText = tag.Label + " \u2705"
	// 	} else {
	// 		buttonText = tag.Label
	// 	}

	// 	row := []tgbotapi.InlineKeyboardButton{
	// 		tgbotapi.NewInlineKeyboardButtonData(buttonText, "TAG_"+strconv.Itoa(int(tag.ID))),
	// 	}
	// 	tagsKeyboard = append(tagsKeyboard, row)
	// }

	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, seasonsKeyboard...)

	// keyboardSubmitCancelGoBack := b.createKeyboard(
	// 	[]string{"Submit - Confirm Changes", "Cancel - clear command", "\U0001F519"},
	// 	[]string{LibrarySeasonsEditSubmitChanges, LibrarySeasonsEditCancel, LibrarySeasonsEditGoBack},
	// )

	// keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, keyboardSubmitCancelGoBack.InlineKeyboard...)

	// Send the message containing season details along with the keyboard
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

func (b *Bot) handleLibrarySeasonsEditToggleMonitor(command *userLibrary) bool {
	command.selectedMonitoring = !command.selectedMonitoring
	b.setLibraryState(command.chatID, command)
	return b.showLibrarySeriesEdit(command)
}

func (b *Bot) handleLibrarySeasonsEditSelectTag(update tgbotapi.Update, command *userLibrary) bool {
	seasonNumberStr := strings.TrimPrefix(update.CallbackQuery.Data, "SEASON_")
	print(seasonNumberStr)
	// // Parse the tag ID
	// tagID, err := strconv.Atoi(tagIDStr)
	// if err != nil {
	// 	fmt.Printf("Cannot convert tag string to int: %v", err)
	// 	return false
	// }

	// // Check if the tag is already selected
	// if isSelectedTag(command.selectedTags, tagID) {
	// 	// If selected, remove the tag from selectedTags (deselect)
	// 	command.selectedTags = removeTag(command.selectedTags, tagID)
	// } else {
	// 	// If not selected, add the tag to selectedTags (select)
	// 	tag := &starr.Tag{ID: tagID} // Create a new starr.Tag with the ID
	// 	command.selectedTags = append(command.selectedTags, tag.ID)
	// }

	b.setLibraryState(command.chatID, command)
	return b.showLibrarySeasons(command)
}

func (b *Bot) handleLibrarySeasonsEditSubmitChanges(update tgbotapi.Update, command *userLibrary) bool {
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
