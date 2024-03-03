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
	AddSeriesTMDBID           = "ADDSERIES_TMDBID_"
	AddSeriesYes              = "ADDSERIES_YES"
	AddSeriesGoBack           = "ADDSERIES_GOBACK"
	AddSeriesProfileGoBack    = "ADDSERIES_QUALITY_GOBACK"
	AddSeriesRootFolderGoBack = "ADDSERIES_ROOTFOLDER_GOBACK"
	AddSeriesTagsGoBack       = "ADDSERIES_TAGSGOBACK"
	AddSeriesTypeGoBack       = "ADDSERIES_TYPEGOBACK"
	AddSeriesAddOptionsGoBack = "ADDSERIES_ADDOPTIONS_GOBACK"
	AddSeriesCancel           = "ADDSERIES_CANCEL"
	AddSeriesTagsDone         = "ADDSERIES_TAGS_DONE"
	AddSeriesMonSea           = "ADDSERIES_MONSEA"
	AddSeriesMon              = "ADDSERIES_MON"
	AddSeriesUnMon            = "ADDSERIES_UNMON"
	AddSeriesColSea           = "ADDSERIES_COLSEA"
	AddSeriesColMon           = "ADDSERIES_COLMON"
)

func (b *Bot) processAddCommand(update tgbotapi.Update, userID int64, s *sonarr.Sonarr) {
	msg := tgbotapi.NewMessage(userID, "Handling add series ommand... please wait")
	message, _ := b.sendMessage(msg)
	command := userAddSeries{
		chatID:    message.Chat.ID,
		messageID: message.MessageID,
	}

	criteria := update.Message.CommandArguments()
	if len(criteria) < 1 {
		b.sendMessageWithEdit(&command, "Please provide a search criteria /q [query]")
		return
	}
	searchResults, err := s.Lookup(criteria)
	if err != nil {
		msg := tgbotapi.NewMessage(userID, err.Error())
		b.sendMessage(msg)
		return
	}

	if len(searchResults) == 0 {
		b.sendMessageWithEdit(&command, "No series found matching your search criteria")
		return
	}
	if len(searchResults) > 25 {
		b.sendMessageWithEdit(&command, "Result size too large, please narrow down your search criteria")
		return
	}

	command.searchResults = make(map[string]*sonarr.Series, len(searchResults))
	for _, series := range searchResults {
		tvdbID := strconv.Itoa(int(series.TvdbID))
		command.searchResults[tvdbID] = series
	}

	b.setAddSeriesState(command.chatID, &command)
	b.setActiveCommand(command.chatID, AddSeriesCommand)
	b.showAddSeriesSearchResults(&command)
}

func (b *Bot) addSeries(update tgbotapi.Update) bool {
	userID, err := b.getUserID(update)
	if err != nil {
		fmt.Printf("Cannot add series: %v", err)
		return false
	}
	command, exists := b.getAddSeriesState(userID)
	if !exists {
		return false
	}
	switch update.CallbackQuery.Data {
	case AddSeriesYes:
		b.setActiveCommand(userID, AddSeriesCommand)
		return b.handleAddSeriesYes(update, command)
	case AddSeriesGoBack:
		b.setAddSeriesState(command.chatID, command)
		return b.showAddSeriesSearchResults(command)
	case AddSeriesProfileGoBack:
		return b.showAddSeriesSearchResults(command)
	case AddSeriesRootFolderGoBack:
		if len(command.allProfiles) == 1 {
			return b.showAddSeriesSearchResults(command)
		}
		return b.showAddSeriesProfiles(command)
	case AddSeriesTagsGoBack:
		if len(command.allRootFolders) == 1 && len(command.allProfiles) == 1 {
			return b.showAddSeriesSearchResults(command)
		}
		if len(command.allRootFolders) == 1 {
			return b.showAddSeriesProfiles(command)
		}
		return b.showAddSeriesRootFolders(command)
	case AddSeriesTypeGoBack:
		// Check if there are no tags
		if len(command.allTags) == 0 {
			// Check if there is only one root folder and one profile
			if len(command.allRootFolders) == 1 && len(command.allProfiles) == 1 {
				return b.showAddSeriesSearchResults(command)
			}
			// Check if there is only one root folder
			if len(command.allRootFolders) == 1 && len(command.allProfiles) > 1 {
				return b.showAddSeriesProfiles(command)
			}
			// Check if there is only one profile
			if len(command.allProfiles) == 1 && len(command.allRootFolders) > 1 {
				return b.showAddSeriesRootFolders(command)
			}
			// If there are multiple root folders and profiles, go to root folders
			return b.showAddSeriesRootFolders(command)
		}
		// If there are tags, go to the tags step
		return b.showAddSeriesTags(command)
	case AddSeriesAddOptionsGoBack:
		return b.showAddSeriesType(command)
	case AddSeriesCancel:
		b.clearState(update)
		b.sendMessageWithEdit(command, CommandsCleared)
		return false
	case AddSeriesTagsDone:
		return b.showAddSeriesType(command)
	case AddSeriesMonSea:
		return b.handleAddSeriesMonSea(update, command)
	case AddSeriesMon:
		return b.handleAddSeriesMon(update, command)
	case AddSeriesUnMon:
		return b.handleAddSeriesUnMon(update, command)
	case AddSeriesColSea:
		return b.handleAddSeriesColSea(update, command)
	case AddSeriesColMon:
		return b.handleAddSeriesColMon(update, command)
	default:
		// Check if it starts with "PROFILE_"
		if strings.HasPrefix(update.CallbackQuery.Data, "PROFILE_") {
			return b.handleAddSeriesProfile(update, command)
		}
		// Check if it starts with "PROFILE_"
		if strings.HasPrefix(update.CallbackQuery.Data, "ROOTFOLDER_") {
			return b.handleAddSeriesRootFolder(update, command)
		}
		// Check if it starts with "TAG_"
		if strings.HasPrefix(update.CallbackQuery.Data, "TAG_") {
			return b.handleAddSeriesEditSelectTag(update, command)
		}
		// Check if it starts with "TYPE_"
		if strings.HasPrefix(update.CallbackQuery.Data, "TYPE_") {
			return b.handleAddSeriesType(update, command)
		}
		// Check if it starts with "ADDSERIES_TMDBID_"
		if strings.HasPrefix(update.CallbackQuery.Data, AddSeriesTMDBID) {
			return b.addSeriesDetails(update, command)
		}
		return b.showAddSeriesSearchResults(command)
	}
}

func (b *Bot) showAddSeriesSearchResults(command *userAddSeries) bool {

	// Extract series from the map
	series := make([]*sonarr.Series, 0, len(command.searchResults))
	for _, s := range command.searchResults {
		series = append(series, s)
	}

	// Sort series by year in ascending order
	sort.SliceStable(series, func(i, j int) bool {
		return series[i].Year < series[j].Year
	})

	var buttonLabels []string
	var buttonData []string
	var text strings.Builder
	var responseText string

	for _, series := range series {
		fmt.Fprintf(&text, "[%v](https://www.imdb.com/title/%v) \\- _%v_\n", utils.Escape(series.Title), series.ImdbID, series.Year)
		buttonLabels = append(buttonLabels, fmt.Sprintf("%v - %v", series.Title, series.Year))
		buttonData = append(buttonData, AddSeriesTMDBID+strconv.Itoa(int(series.TvdbID)))
	}

	keyboard := b.createKeyboard(buttonLabels, buttonData)
	keyboardCancel := b.createKeyboard(
		[]string{"Cancel - clear command"},
		[]string{AddSeriesCancel},
	)
	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, keyboardCancel.InlineKeyboard...)

	switch len(command.searchResults) {
	case 1:
		responseText = "*Series found*\n\n"
	default:
		responseText = fmt.Sprintf("*Found %d series*\n\n", len(command.searchResults))
	}
	responseText += text.String()

	editMsg := tgbotapi.NewEditMessageTextAndMarkup(
		command.chatID,
		command.messageID,
		responseText,
		keyboard,
	)
	editMsg.ParseMode = "MarkdownV2"
	editMsg.DisableWebPagePreview = true
	b.setAddSeriesState(command.chatID, command)
	b.sendMessage(editMsg)
	return false
}

func (b *Bot) addSeriesDetails(update tgbotapi.Update, command *userAddSeries) bool {
	seriesIDStr := strings.TrimPrefix(update.CallbackQuery.Data, AddSeriesTMDBID)
	command.series = command.searchResults[seriesIDStr]

	var text strings.Builder
	fmt.Fprintf(&text, "Is this the correct series?\n\n")
	fmt.Fprintf(&text, "[%v](https://www.imdb.com/title/%v) \\- _%v_\n\n", utils.Escape(command.series.Title), command.series.ImdbID, command.series.Year)

	keyboard := b.createKeyboard(
		[]string{"Yes, add this series", "\U0001F519"},
		[]string{AddSeriesYes, AddSeriesGoBack})

	editMsg := tgbotapi.NewEditMessageTextAndMarkup(
		command.chatID,
		command.messageID,
		text.String(),
		keyboard,
	)
	editMsg.ParseMode = "MarkdownV2"
	editMsg.DisableWebPagePreview = false
	b.setAddSeriesState(command.chatID, command)
	b.sendMessage(editMsg)
	return false
}

func (b *Bot) handleAddSeriesYes(update tgbotapi.Update, command *userAddSeries) bool {
	//series already in library...
	if command.series.ID != 0 {
		b.sendMessageWithEdit(command, "Series already in library\nAll commands have been cleared")
		return false
	}

	profiles, err := b.SonarrServer.GetQualityProfiles()
	if err != nil {
		msg := tgbotapi.NewMessage(command.chatID, err.Error())
		fmt.Println(err)
		b.sendMessage(msg)
		return false
	}
	if len(profiles) == 0 {
		b.sendMessageWithEdit(command, "No quality profile(s) found on your sonarr server.\nAll commands have been cleared.")
		b.clearState(update)
	}
	if len(profiles) == 1 {
		command.profileID = profiles[0].ID
	}
	command.allProfiles = profiles

	rootFolders, err := b.SonarrServer.GetRootFolders()
	if err != nil {
		msg := tgbotapi.NewMessage(command.chatID, err.Error())
		fmt.Println(err)
		b.sendMessage(msg)
		return false
	}
	if len(rootFolders) == 1 {
		command.rootFolder = rootFolders[0].Path
	}
	if len(rootFolders) == 0 {
		b.sendMessageWithEdit(command, "No root folder(s) found on your radarr server.\nAll commands have been cleared.")
		b.clearState(update)
	}
	command.allRootFolders = rootFolders

	tags, err := b.SonarrServer.GetTags()
	if err != nil {
		msg := tgbotapi.NewMessage(command.chatID, err.Error())
		fmt.Println(err)
		b.sendMessage(msg)
		return false
	}
	command.allTags = tags

	b.setAddSeriesState(command.chatID, command)
	return b.showAddSeriesProfiles(command)
}

func (b *Bot) showAddSeriesProfiles(command *userAddSeries) bool {
	// If there is only one profile, skip this step
	if len(command.allProfiles) == 1 {
		return b.showAddSeriesRootFolders(command)
	}
	var profileKeyboard [][]tgbotapi.InlineKeyboardButton
	for _, profile := range command.allProfiles {
		row := []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(profile.Name, "PROFILE_"+strconv.Itoa(int(profile.ID))),
		}
		profileKeyboard = append(profileKeyboard, row)
	}

	var messageText strings.Builder
	var keyboard tgbotapi.InlineKeyboardMarkup
	keyboardGoBack := b.createKeyboard(
		[]string{"\U0001F519"},
		[]string{AddSeriesProfileGoBack},
	)
	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, profileKeyboard...)
	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, keyboardGoBack.InlineKeyboard...)
	messageText.WriteString("Select quality profile:")
	b.sendMessageWithEditAndKeyboard(
		command,
		keyboard,
		messageText.String(),
	)
	return false
}

func (b *Bot) handleAddSeriesProfile(update tgbotapi.Update, command *userAddSeries) bool {
	profileIDStr := strings.TrimPrefix(update.CallbackQuery.Data, "PROFILE_")
	// Parse the profile ID
	profileID, err := strconv.Atoi(profileIDStr)
	if err != nil {
		msg := tgbotapi.NewMessage(command.chatID, err.Error())
		fmt.Println(err)
		b.sendMessage(msg)
		return false
	}
	command.profileID = int64(profileID)
	b.setAddSeriesState(command.chatID, command)
	return b.showAddSeriesRootFolders(command)
}

func (b *Bot) showAddSeriesRootFolders(command *userAddSeries) bool {
	// If there is only one root folder, skip this step
	if len(command.allRootFolders) == 1 {
		return b.showAddSeriesTags(command)
	}
	var rootFolderKeyboard [][]tgbotapi.InlineKeyboardButton
	for _, rootFolder := range command.allRootFolders {
		row := []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(rootFolder.Path, "ROOTFOLDER_"+rootFolder.Path),
		}
		rootFolderKeyboard = append(rootFolderKeyboard, row)
	}

	var messageText strings.Builder
	var keyboard tgbotapi.InlineKeyboardMarkup
	keyboardGoBack := b.createKeyboard(
		[]string{"\U0001F519"},
		[]string{AddSeriesRootFolderGoBack},
	)
	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, rootFolderKeyboard...)
	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, keyboardGoBack.InlineKeyboard...)
	messageText.WriteString("Select root folder:")
	b.sendMessageWithEditAndKeyboard(
		command,
		keyboard,
		messageText.String(),
	)
	return false

}

func (b *Bot) handleAddSeriesRootFolder(update tgbotapi.Update, command *userAddSeries) bool {
	command.rootFolder = strings.TrimPrefix(update.CallbackQuery.Data, "ROOTFOLDER_")
	b.setAddSeriesState(command.chatID, command)
	return b.showAddSeriesTags(command)
}

func (b *Bot) showAddSeriesTags(command *userAddSeries) bool {
	// If there are no tags, skip this step
	if len(command.allTags) == 0 {
		return b.showAddSeriesAddOptions(command)
	}
	var tagsKeyboard [][]tgbotapi.InlineKeyboardButton
	for _, tag := range command.allTags {
		// Check if the tag is selected
		isSelected := isSelectedTag(command.selectedTags, tag.ID)

		buttonText := tag.Label
		if isSelected {
			buttonText += " \u2705"
		}

		row := []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(buttonText, "TAG_"+strconv.Itoa(int(tag.ID))),
		}
		tagsKeyboard = append(tagsKeyboard, row)
	}
	var keyboard tgbotapi.InlineKeyboardMarkup
	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, tagsKeyboard...)

	keyboardSubmitCancelGoBack := b.createKeyboard(
		[]string{"Done - Continue", "\U0001F519"},
		[]string{AddSeriesTagsDone, AddSeriesTagsGoBack},
	)

	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, keyboardSubmitCancelGoBack.InlineKeyboard...)

	// Send the message containing series details along with the keyboard
	editMsg := tgbotapi.NewEditMessageTextAndMarkup(
		command.chatID,
		command.messageID,
		"Select tags:",
		keyboard,
	)
	editMsg.ParseMode = "MarkdownV2"
	editMsg.DisableWebPagePreview = true
	b.setAddSeriesState(command.chatID, command)
	b.sendMessage(editMsg)
	return false

}

func (b *Bot) handleAddSeriesEditSelectTag(update tgbotapi.Update, command *userAddSeries) bool {
	tagIDStr := strings.TrimPrefix(update.CallbackQuery.Data, "TAG_")
	// Parse the tag ID
	tagID, err := strconv.Atoi(tagIDStr)
	if err != nil {
		fmt.Printf("Cannot convert tag string to int: %v", err)
		return false
	}
	// Check if the tag is already selected
	if isSelectedTag(command.selectedTags, tagID) {
		// If selected, remove the tag from selectedTags (deselect)
		command.selectedTags = removeTag(command.selectedTags, tagID)
	} else {
		// If not selected, add the tag to selectedTags (select)
		tag := &starr.Tag{ID: tagID} // Create a new starr.Tag with the ID
		command.selectedTags = append(command.selectedTags, tag.ID)
	}

	b.setAddSeriesState(command.chatID, command)
	return b.showAddSeriesTags(command)
}

func (b *Bot) showAddSeriesType(command *userAddSeries) bool {

	types := []string{"standard", "daily", "anime"}
	var typeKeyboard [][]tgbotapi.InlineKeyboardButton
	for _, t := range types {
		row := []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(t, "TYPE_"+strings.ToUpper(t)),
		}
		typeKeyboard = append(typeKeyboard, row)
	}

	var messageText strings.Builder
	var keyboard tgbotapi.InlineKeyboardMarkup
	keyboardGoBack := b.createKeyboard(
		[]string{"\U0001F519"},
		[]string{AddSeriesTypeGoBack},
	)
	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, typeKeyboard...)
	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, keyboardGoBack.InlineKeyboard...)
	messageText.WriteString("Select series type:")
	b.sendMessageWithEditAndKeyboard(
		command,
		keyboard,
		messageText.String(),
	)
	return false

}

func (b *Bot) handleAddSeriesType(update tgbotapi.Update, command *userAddSeries) bool {
	command.seriesType = strings.ToLower(strings.TrimPrefix(update.CallbackQuery.Data, "TYPE_"))
	b.setAddSeriesState(command.chatID, command)
	return b.showAddSeriesAddOptions(command)
}

func (b *Bot) showAddSeriesAddOptions(command *userAddSeries) bool {
	keyboard := b.createKeyboard(
		[]string{"Add series monitored + search now", "Add series monitored", "Add series unmonitored", "Add collection monitored + search now", "Add collection monitored", "Cancel, clear command", "\U0001F519"},
		[]string{AddSeriesMonSea, AddSeriesMon, AddSeriesUnMon, AddSeriesColSea, AddSeriesColMon, AddSeriesCancel, AddSeriesAddOptionsGoBack},
	)
	editMsg := tgbotapi.NewEditMessageTextAndMarkup(
		command.chatID,
		command.messageID,
		"How would you like to add the series?\n",
		keyboard,
	)
	editMsg.ParseMode = "MarkdownV2"
	editMsg.DisableWebPagePreview = true
	b.setAddSeriesState(command.chatID, command)
	b.sendMessage(editMsg)
	return false
}

func (b *Bot) handleAddSeriesMonSea(update tgbotapi.Update, command *userAddSeries) bool {
	command.monitored = *starr.True()
	command.addSeriesOptions = &sonarr.AddSeriesOptions{
		SearchForMissingEpisodes: *starr.True(),
		//Monitor:                  "seriesOnly",
	}
	b.setAddSeriesState(command.chatID, command)
	return b.addSeriesToLibrary(update, command)
}

func (b *Bot) handleAddSeriesMon(update tgbotapi.Update, command *userAddSeries) bool {
	command.monitored = *starr.True()
	command.addSeriesOptions = &sonarr.AddSeriesOptions{
		SearchForMissingEpisodes: *starr.False(),
		//Monitor:                  "seriesOnly",
	}
	b.setAddSeriesState(command.chatID, command)
	return b.addSeriesToLibrary(update, command)
}

func (b *Bot) handleAddSeriesUnMon(update tgbotapi.Update, command *userAddSeries) bool {
	command.monitored = *starr.False()
	command.addSeriesOptions = &sonarr.AddSeriesOptions{
		SearchForMissingEpisodes: *starr.False(),
		//Monitor:        "none",
	}
	b.setAddSeriesState(command.chatID, command)
	return b.addSeriesToLibrary(update, command)
}

func (b *Bot) handleAddSeriesColSea(update tgbotapi.Update, command *userAddSeries) bool {
	command.monitored = *starr.True()
	command.addSeriesOptions = &sonarr.AddSeriesOptions{
		SearchForMissingEpisodes: *starr.True(),
		//Monitor:        "seriesAndCollection",
	}
	b.setAddSeriesState(command.chatID, command)
	return b.addSeriesToLibrary(update, command)
}

func (b *Bot) handleAddSeriesColMon(update tgbotapi.Update, command *userAddSeries) bool {
	command.monitored = *starr.True()
	command.addSeriesOptions = &sonarr.AddSeriesOptions{
		SearchForMissingEpisodes: *starr.False(),
		//Monitor:                  "seriesAndCollection",
	}
	b.setAddSeriesState(command.chatID, command)
	return b.addSeriesToLibrary(update, command)
}

func (b *Bot) addSeriesToLibrary(update tgbotapi.Update, command *userAddSeries) bool {
	var tagIDs []int
	tagIDs = append(tagIDs, command.selectedTags...)

	// does anyone ever user anything other than announced?
	addSeriesInput := sonarr.AddSeriesInput{
		//MinimumAvailability: "announced",
		TvdbID:           command.series.TvdbID,
		Title:            command.series.Title,
		QualityProfileID: command.profileID,
		RootFolderPath:   command.rootFolder,
		AddOptions:       command.addSeriesOptions,
		Tags:             tagIDs,
		Monitored:        command.monitored,
	}

	var messageText string
	var _, err = b.SonarrServer.AddSeries(&addSeriesInput)
	if err != nil {
		msg := tgbotapi.NewMessage(command.chatID, err.Error())
		fmt.Println(err)
		b.sendMessage(msg)
		return false
	}
	// series, err := b.SonarrServer.GetSeries((command.series.TvdbID))
	// if err != nil {
	// 	msg := tgbotapi.NewMessage(command.chatID, err.Error())
	// 	fmt.Println(err)
	// 	b.sendMessage(msg)
	// 	return false
	// }

	// if command.addSeriesOptions.Monitor == "seriesAndCollection" {
	// 	messageText = fmt.Sprintf("Collection '%v' added\n", series[0].Title)
	// } else {
	// 	messageText = fmt.Sprintf("Series'%v' added\n", series[0].Title)
	// }
	b.sendMessageWithEdit(command, messageText)
	b.clearState(update)
	return true
}
