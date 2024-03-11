module github.com/woiza/telegram-bot-sonarr

go 1.22.1

require (
	github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.5.1
	golift.io/starr v1.0.0
)

require golang.org/x/net v0.22.0 // indirect

replace golift.io/starr v1.0.0 => github.com/woiza/starr v0.0.0-20240311094657-abe7f063757b
