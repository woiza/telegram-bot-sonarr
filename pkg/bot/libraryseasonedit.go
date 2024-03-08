package bot

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/woiza/telegram-bot-sonarr/pkg/utils"
	"golift.io/starr"
)

const (
	LibrarySeasonEditToggleMonitor = "LIBRARY_SEASON_EDIT_TOGGLE_MONITOR"
	LibrarySeasonEditSubmitChanges = "LIBRARY_SEASON_EDIT_SUBMIT_CHANGES"
	LibrarySeasonEditGoBack        = "LIBRARY_SEASON_EDIT_GOBACK"
	LibrarySeasonEditCancel        = "LIBRARY_SEASON_EDIT_CANCEL"
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
		row := []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(strconv.Itoa(season.SeasonNumber), "SEASON_"+strconv.Itoa(int(season.SeasonNumber))),
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
	return b.showLibrarySeason(command)
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
