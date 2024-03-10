package bot

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"golift.io/starr"
	"golift.io/starr/sonarr"
)

func (b *Bot) sendUpcoming(episodes []*sonarr.Episode, msg *tgbotapi.MessageConfig) {
	sort.SliceStable(episodes, func(i, j int) bool {
		return episodes[i].AirDateUtc.Before(episodes[j].AirDateUtc)
	})

	seriesMap := make(map[int64]*sonarr.Series)

	for i := 0; i < len(episodes); i += b.Config.MaxItems {
		end := i + b.Config.MaxItems
		if end > len(episodes) {
			end = len(episodes)
		}

		var text strings.Builder
		for _, episode := range episodes[i:end] {
			series, ok := seriesMap[episode.SeriesID]
			if !ok {
				var err error
				series, err = b.SonarrServer.GetSeriesByID(episode.SeriesID)
				if err != nil {
					msg.Text = err.Error()
					b.sendMessage(msg)
					return
				}
				seriesMap[episode.SeriesID] = series
			}

			fmt.Fprintf(&text, "[%v](https://www.imdb.com/title/%v) %vx%02d \\- %v\n", series.Title, series.ImdbID, episode.SeasonNumber, episode.EpisodeNumber, episode.AirDateUtc.Format("02 Jan 2006"))
		}

		msg.Text = text.String()
		msg.ParseMode = "MarkdownV2"
		msg.DisableWebPagePreview = true
		b.sendMessage(msg)
	}
}

func (b *Bot) getSeriesAsInlineKeyboard(series []*sonarr.Series) [][]tgbotapi.InlineKeyboardButton {
	var inlineKeyboard [][]tgbotapi.InlineKeyboardButton
	for _, series := range series {
		button := tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("%v - %v", series.Title, series.Year),
			"TVDBID_"+strconv.Itoa(int(series.TvdbID)),
		)
		row := []tgbotapi.InlineKeyboardButton{button}
		inlineKeyboard = append(inlineKeyboard, row)
	}
	return inlineKeyboard
}

func (b *Bot) createKeyboard(buttonText, buttonData []string) tgbotapi.InlineKeyboardMarkup {
	buttons := make([][]tgbotapi.InlineKeyboardButton, len(buttonData))
	for i := range buttonData {
		buttons[i] = tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(buttonText[i], buttonData[i]))
	}
	return tgbotapi.NewInlineKeyboardMarkup(buttons...)
}

func findTagByID(tags []*starr.Tag, tagID int) *starr.Tag {
	for _, tag := range tags {
		if int(tag.ID) == tagID {
			return tag
		}
	}
	return nil
}

func isSelectedTag(selectedTags []int, tagID int) bool {
	for _, selectedTag := range selectedTags {
		if selectedTag == tagID {
			return true
		}
	}
	return false
}

func removeTag(tags []int, tagID int) []int {
	var updatedTags []int
	for _, tag := range tags {
		if tag != tagID {
			updatedTags = append(updatedTags, tag)
		}
	}
	return updatedTags
}

func getQualityProfileByID(qualityProfiles []*sonarr.QualityProfile, qualityProfileID int64) *sonarr.QualityProfile {
	for _, profile := range qualityProfiles {
		if profile.ID == qualityProfileID {
			return profile
		}
	}
	return nil
}

func getQualityProfileIndexByID(qualityProfiles []*sonarr.QualityProfile, qualityProfileId int64) int {
	for i, profile := range qualityProfiles {
		if profile.ID == qualityProfileId {
			return i
		}
	}
	return -1 // Return an appropriate default or handle the error as needed
}

func getSeasonByNumber(series *sonarr.Series, number int) *sonarr.Season {
	for i := range series.Seasons {
		if series.Seasons[i].SeasonNumber == number {
			return series.Seasons[i]
		}
	}
	return nil
}

func seriesToAddSeriesInput(series *sonarr.Series) *sonarr.AddSeriesInput {
	return &sonarr.AddSeriesInput{
		Monitored:         series.Monitored,
		SeasonFolder:      series.SeasonFolder,
		UseSceneNumbering: series.UseSceneNumbering,
		ID:                series.ID,
		LanguageProfileID: series.LanguageProfileID,
		QualityProfileID:  series.QualityProfileID,
		TvdbID:            series.TvdbID,
		ImdbID:            series.ImdbID,
		TvMazeID:          series.TvMazeID,
		TvRageID:          series.TvRageID,
		Path:              series.Path,
		SeriesType:        series.SeriesType,
		Title:             series.Title,
		TitleSlug:         series.TitleSlug,
		RootFolderPath:    series.RootFolderPath,
		Tags:              series.Tags,
		Seasons:           series.Seasons,
		Images:            series.Images,
	}
}
