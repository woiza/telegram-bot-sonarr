package bot

// import (
// 	"fmt"
// 	"sort"
// 	"strconv"
// 	"strings"

// 	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
// 	"github.com/woiza/telegram-bot-sonarr/pkg/utils"
// 	"golift.io/starr"
// 	"golift.io/starr/sonarr"
// )

// func (b *Bot) sendUpcoming(series []*sonarr.Episode, msg *tgbotapi.MessageConfig) {
// 	sort.SliceStable(series, func(i, j int) bool {
// 		return utils.IgnoreArticles(strings.ToLower(series[i].Title)) < utils.IgnoreArticles(strings.ToLower(series[j].Title))
// 	})
// 	for i := 0; i < len(series); i += b.Config.MaxItems {
// 		end := i + b.Config.MaxItems
// 		if end > len(series) {
// 			end = len(series)
// 		}

// 		//TODO
// 		//rewrite this for sonarr series
// 		var text strings.Builder
// 		for _, series := range series[i:end] {
// 			if !series.FirstAired.IsZero() {
// 				fmt.Fprintf(&text, "[%v](https://www.imdb.com/title/%v) \\- first aired %v\n", utils.Escape(series.Title), series.ImdbID, utils.Escape(series.FirstAired.Format("02 Jan 2006")))
// 			}
// 			if !series.NextAiring.IsZero() {
// 				fmt.Fprintf(&text, "[%v](https://www.imdb.com/title/%v) \\- next airing %v\n", utils.Escape(series.Title), series.ImdbID, utils.Escape(series.NextAiring.Format("02 Jan 2006")))
// 			}
// 		}

// 		msg.Text = text.String()
// 		msg.ParseMode = "MarkdownV2"
// 		msg.DisableWebPagePreview = true
// 		b.sendMessage(msg)
// 	}
// }

// func (b *Bot) getSeriesAsInlineKeyboard(series []*sonarr.Series) [][]tgbotapi.InlineKeyboardButton {
// 	var inlineKeyboard [][]tgbotapi.InlineKeyboardButton
// 	for _, series := range series {
// 		button := tgbotapi.NewInlineKeyboardButtonData(
// 			fmt.Sprintf("%v - %v", series.Title, series.Year),
// 			"TVDBID_"+strconv.Itoa(int(series.TvdbID)),
// 		)
// 		row := []tgbotapi.InlineKeyboardButton{button}
// 		inlineKeyboard = append(inlineKeyboard, row)
// 	}
// 	return inlineKeyboard
// }

// func (b *Bot) createKeyboard(buttonText, buttonData []string) tgbotapi.InlineKeyboardMarkup {
// 	buttons := make([][]tgbotapi.InlineKeyboardButton, len(buttonData))
// 	for i := range buttonData {
// 		buttons[i] = tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(buttonText[i], buttonData[i]))
// 	}
// 	return tgbotapi.NewInlineKeyboardMarkup(buttons...)
// }

// func findTagByID(tags []*starr.Tag, tagID int) *starr.Tag {
// 	for _, tag := range tags {
// 		if int(tag.ID) == tagID {
// 			return tag
// 		}
// 	}
// 	return nil
// }

// func isSelectedTag(selectedTags []int, tagID int) bool {
// 	for _, selectedTag := range selectedTags {
// 		if selectedTag == tagID {
// 			return true
// 		}
// 	}
// 	return false
// }

// func removeTag(tags []int, tagID int) []int {
// 	var updatedTags []int
// 	for _, tag := range tags {
// 		if tag != tagID {
// 			updatedTags = append(updatedTags, tag)
// 		}
// 	}
// 	return updatedTags
// }
