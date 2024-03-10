package bot

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"golift.io/starr"
	"golift.io/starr/sonarr"

	"github.com/woiza/telegram-bot-sonarr/pkg/config"
)

const (
	AddSeriesCommand          = "ADDSERIES"
	DeleteSeriesCommand       = "DELETESERIES"
	LibraryMenuCommand        = "LIBRARYMENU"
	LibraryFilteredCommand    = "LIBRARYFILTERED"
	LibrarySeriesEditCommand  = "LIBRARYSERIESEDIT"
	LibrarySeasonsEditCommand = "LIBRARYSEASONSEDIT"
	CommandsClearedMessage    = "I am not sure what you mean.\nAll commands have been cleared"
	CommandsCleared           = "All commands have been cleared"
)

type userAddSeries struct {
	searchResults    map[string]*sonarr.Series
	series           *sonarr.Series
	allProfiles      []*sonarr.QualityProfile
	profileID        int64
	allRootFolders   []*sonarr.RootFolder
	rootFolder       string
	allTags          []*starr.Tag
	selectedTags     []int
	seriesType       string
	monitor          string
	addSeriesOptions *sonarr.AddSeriesOptions
	chatID           int64
	messageID        int
}

type userDeleteSeries struct {
	library            map[string]*sonarr.Series
	seriesForSelection []*sonarr.Series // Series to select from, either whole library or search results
	selectedSeries     []*sonarr.Series
	chatID             int64
	messageID          int
	page               int
}

type userLibrary struct {
	library                []*sonarr.Series
	libraryFiltered        map[string]*sonarr.Series
	searchResultsInLibrary []*sonarr.Series
	filter                 string
	qualityProfiles        []*sonarr.QualityProfile
	selectedQualityProfile int64
	allTags                []*starr.Tag
	selectedTags           []int
	selectedMonitoring     bool
	series                 *sonarr.Series
	seriesSeasons          map[int]*sonarr.Season
	selectedSeason         int
	lastSeriesSearch       time.Time
	lastSeasonSearch       time.Time
	chatID                 int64
	messageID              int
	page                   int
}

type Bot struct {
	Config             *config.Config
	Bot                *tgbotapi.BotAPI
	SonarrServer       *sonarr.Sonarr
	ActiveCommand      map[int64]string
	AddSeriesStates    map[int64]*userAddSeries
	DeleteSeriesStates map[int64]*userDeleteSeries
	LibraryStates      map[int64]*userLibrary
	// Mutexes for synchronization
	muActiveCommand      sync.Mutex
	muAddSeriesStates    sync.Mutex
	muDeleteSeriesStates sync.Mutex
	muLibraryStates      sync.Mutex
}

type Command interface {
	GetChatID() int64
	GetMessageID() int
}

// Implement the interface for userLibrary
func (c *userLibrary) GetChatID() int64 {
	return c.chatID
}

func (c *userLibrary) GetMessageID() int {
	return c.messageID
}

// Implement the interface for userDelete
func (c *userDeleteSeries) GetChatID() int64 {
	return c.chatID
}

func (c *userDeleteSeries) GetMessageID() int {
	return c.messageID
}

// Implement the interface for userAddSeries
func (c *userAddSeries) GetChatID() int64 {
	return c.chatID
}

func (c *userAddSeries) GetMessageID() int {
	return c.messageID
}

func New(config *config.Config, botAPI *tgbotapi.BotAPI, sonarrServer *sonarr.Sonarr) *Bot {
	return &Bot{
		Config:             config,
		Bot:                botAPI,
		SonarrServer:       sonarrServer,
		ActiveCommand:      make(map[int64]string),
		AddSeriesStates:    make(map[int64]*userAddSeries),
		DeleteSeriesStates: make(map[int64]*userDeleteSeries),
		LibraryStates:      make(map[int64]*userLibrary),
	}
}

func (b *Bot) HandleUpdates(updates <-chan tgbotapi.Update) {
	for update := range updates {
		b.HandleUpdate(update)
	}
}

func (b *Bot) HandleUpdate(update tgbotapi.Update) {
	userID, err := b.getUserID(update)
	if err != nil {
		fmt.Printf("Cannot handle update: %v", err)
		return
	}

	if update.Message != nil && !b.Config.AllowedUserIDs[userID] {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Access denied. You are not authorized.")
		b.sendMessage(msg)
		return
	}

	activeCommand, _ := b.getActiveCommand(userID)

	if update.CallbackQuery != nil {
		switch activeCommand {
		case AddSeriesCommand:
			if !b.addSeries(update) {
				return
			}
		case DeleteSeriesCommand:
			if !b.deleteSeries(update) {
				return
			}
		case LibraryMenuCommand:
			if !b.libraryMenu(update) {
				return
			}
		case LibraryFilteredCommand:
			if !b.libraryFiltered(update) {
				return
			}
		case LibrarySeriesEditCommand:
			if !b.librarySeriesEdit(update) {
				return
			}
		case LibrarySeasonsEditCommand:
			if !b.librarySeasonEdit(update) {
				return
			}
		default:
			b.clearState(update)
			msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, CommandsClearedMessage)
			b.sendMessage(msg)
		}
	}
	if update.Message == nil { // ignore any non-Message Updates
		return
	}

	// If no command was passed, handle a search command.
	if update.Message.Entities == nil {
		//update.Message.Text = fmt.Sprintf("/q \"%s\"", update.Message.Text)
		update.Message.Text = fmt.Sprintf("/q %s", update.Message.Text)
		update.Message.Entities = []tgbotapi.MessageEntity{{
			Type:   "bot_command",
			Length: 2, // length of the command `/q`
		}}
	}

	if update.Message.IsCommand() {
		b.handleCommand(update, b.SonarrServer)
	}
}

func (b *Bot) clearState(update tgbotapi.Update) {
	userID, err := b.getUserID(update)
	if err != nil {
		fmt.Printf("Cannot clear state: %v", err)
		return
	}

	// Safely clear states using mutexes
	b.muActiveCommand.Lock()
	defer b.muActiveCommand.Unlock()

	delete(b.ActiveCommand, userID)

	b.muAddSeriesStates.Lock()
	defer b.muAddSeriesStates.Unlock()

	delete(b.AddSeriesStates, userID)

	b.muDeleteSeriesStates.Lock()
	defer b.muDeleteSeriesStates.Unlock()

	delete(b.DeleteSeriesStates, userID)

	b.muLibraryStates.Lock()
	defer b.muLibraryStates.Unlock()

	delete(b.LibraryStates, userID)
}

func (b *Bot) getUserID(update tgbotapi.Update) (int64, error) {
	var userID int64
	if update.Message != nil {
		userID = update.Message.From.ID
	}
	if update.CallbackQuery != nil {
		userID = update.CallbackQuery.From.ID
	}
	if userID == 0 {
		return 0, fmt.Errorf("no user ID found in Message and CallbackQuery")
	}
	return userID, nil
}

func (b *Bot) getActiveCommand(userID int64) (string, bool) {
	b.muActiveCommand.Lock()
	defer b.muActiveCommand.Unlock()
	cmd, exists := b.ActiveCommand[userID]
	return cmd, exists
}

func (b *Bot) setActiveCommand(userID int64, command string) {
	b.muActiveCommand.Lock()
	defer b.muActiveCommand.Unlock()
	b.ActiveCommand[userID] = command
}

func (b *Bot) getAddSeriesState(userID int64) (*userAddSeries, bool) {
	b.muAddSeriesStates.Lock()
	defer b.muAddSeriesStates.Unlock()
	state, exists := b.AddSeriesStates[userID]
	return state, exists
}

func (b *Bot) setAddSeriesState(userID int64, state *userAddSeries) {
	b.muAddSeriesStates.Lock()
	defer b.muAddSeriesStates.Unlock()
	b.AddSeriesStates[userID] = state
}

func (b *Bot) getDeleteSeriesState(userID int64) (*userDeleteSeries, bool) {
	b.muDeleteSeriesStates.Lock()
	defer b.muDeleteSeriesStates.Unlock()
	state, exists := b.DeleteSeriesStates[userID]
	return state, exists
}

func (b *Bot) setDeleteSeriesState(userID int64, state *userDeleteSeries) {
	b.muDeleteSeriesStates.Lock()
	defer b.muDeleteSeriesStates.Unlock()
	b.DeleteSeriesStates[userID] = state
}

func (b *Bot) getLibraryState(userID int64) (*userLibrary, bool) {
	b.muLibraryStates.Lock()
	defer b.muLibraryStates.Unlock()
	state, exists := b.LibraryStates[userID]
	return state, exists
}

func (b *Bot) setLibraryState(userID int64, state *userLibrary) {
	b.muLibraryStates.Lock()
	defer b.muLibraryStates.Unlock()
	b.LibraryStates[userID] = state
}

func (b *Bot) sendMessage(msg tgbotapi.Chattable) (tgbotapi.Message, error) {
	message, err := b.Bot.Send(msg)
	if err != nil {
		log.Println("Error sending message:", err)
	}
	return message, err
}

func (b *Bot) sendMessageWithEdit(command Command, text string) {
	editMsg := tgbotapi.NewEditMessageText(
		command.GetChatID(),
		command.GetMessageID(),
		text,
	)
	_, err := b.sendMessage(editMsg)
	if err != nil {
		log.Printf("Error editing message: %v", err)
	}
}

func (b *Bot) sendMessageWithEditAndKeyboard(command Command, keyboard tgbotapi.InlineKeyboardMarkup, text string) {
	editMsg := tgbotapi.NewEditMessageTextAndMarkup(
		command.GetChatID(),
		command.GetMessageID(),
		text,
		keyboard,
	)
	_, err := b.sendMessage(editMsg)
	if err != nil {
		log.Printf("Error editing message with keyboard: %v", err)
	}
}

func prettyPrint(i interface{}) string {
	s, _ := json.MarshalIndent(i, "", "\t")
	return string(s)
}
