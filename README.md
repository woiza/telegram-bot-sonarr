# Go-Powered Telegram Bot for Sonarr Series Management
This Telegram bot is specifically designed for series management through Sonarr. It enables users to execute a range of commands for searching, adding, editing, deleting, and organizing series within their Sonarr library. Developed in Go, the bot operates with minimal resource consumption, utilizing less than 10 MB of RAM. It maintains a stateless operation and does not persist data to disk, except for error logs. The Docker image size is efficiently kept under 10 MB (compressed), supporting multiple CPU architectures including `arm32v7`, `arm64v8`, and `x86_64`/`amd64`.

This bot is built using [golift/starr](https://github.com/golift/starr/) and [go-telegram-bot-api/telegram-bot-api](https://github.com/go-telegram-bot-api/telegram-bot-api/) without any additional dependencies.

## Features and Commands

<img src="screenshots/menu.png?raw=true" alt="menu" title="menu" width="300" />

### Start Bot
<img src="screenshots/start.png?raw=true" alt="start" title="start" width="300" />

### Search and Add Series
``/q [series]`` or just type the series's title: Search for a series.\
Once a series is found, the bot offers options to add the series to your Sonarr library along with various monitoring settings. If you have only one root folder and one quality profile, the bot will automatically select the first option for you. However, if multiple choices exist, you will be prompted to select a root folder and a quality profile. If you have tags defined in Sonarr, you can select them as well.

<img src="screenshots/add_links.png?raw=true" alt="qlinks" title="add series" width="300" />
<img src="screenshots/add_confirmation.png?raw=true" alt="qconfirmation" title="add confirmation" width="300" />
<img src="screenshots/add_monitor.png?raw=true" alt="qmonitoring" title="add monitoring" width="300" />
<img src="screenshots/add_search.png?raw=true" alt="qsearch" title="add search" width="300" />

### Series Management
``/library [series]`` or ``/l [series]``: Manage series in your library. Allows editing a series' quality profile (if more than one is configured in Sonarr) and tags. Furthermore, you can monitor/unmonitor a series, delete it, search for it, edit/delete/search its seasons, and see disk usage. Series/title is optional. If omitted, a filter menu is shown.

<img src="screenshots/library.png?raw=true" alt="l" title="library" width="300" />
<img src="screenshots/library_series.png?raw=true" alt="lseries" title="library series" width="300" />
<img src="screenshots/library_seasons.png?raw=true" alt="lseasons" title="library seasons" width="300" />
<img src="screenshots/library_season.png?raw=true" alt="lseason" title="library season" width="300" />



### Series Deletion
``/delete [series]`` or ``/d [series]``: Initiate the process of deleting a or several series from your Sonarr library. Be cautious as this action deletes associated files. Series/title is optional. If omitted, all series are shown as inline keyboards and multiple series can be selected.

<img src="screenshots/delete_confirmation.png?raw=true" alt="q1" title="delete" width="300" />

### Cancel or Abort Commands
``/clear`` or ``/cancel`` or ``/stop``: 
This command clears all previously issued commands and resets the bot's state. It can be issued at any time.

### Library Management
- ``/up`` or ``/upcoming``: List upcoming episodes in the next 30 days
- ``/rss``: Initiate an RSS sync
<!---
- ``/searchmonitored``: Search all monitored series/episodes
- ``/updateall``: Update metadata and rescan files/folders for all series/episodes
-->

### System Information
- ``/free`` or ``/diskspace``: Display free space of disks connected to your Sonarr server
- ``/system`` : Display your Sonarr configuration
- ``/id`` or ``/getid``: Show your Telegram user ID


## Installation and Configuration
You can either build the bot yourself using the provided source code or utilize the Docker image hosted on GitHub Container Registry and Docker Hub:
- GitHub [ghcr.io/woiza/telegram-bot-sonarr](https://github.com/woiza/telegram-bot-sonarr/pkgs/container/telegram-bot-sonarr)
- Docker Hub [woiza/telegram-bot-sonarr](https://hub.docker.com/repository/docker/woiza/telegram-bot-sonarr/)

The bot requires configuration through seven mandatory environment variables. For specific details, please refer to the Docker Compose example provided below. Before running this bot, ensure you have obtained a Telegram bot token and your Sonarr API key. Additionally, determine who should have access to this bot (Telegram user ID). Several users are supported by providing a list of Telegram user IDs. You can find detailed instructions on obtaining these credentials in the official documentation:
- [Telegram Bot Token](https://core.telegram.org/bots/tutorial/)
- [Sonarr API Key](https://wiki.servarr.com/en/sonarr/settings#security/)



### Build Docker Image
```
docker buildx build --push --platform linux/amd64,linux/arm64,linux/arm/v7 --tag <repo>/<image>:<tag> .
```


### Docker Compose Example
```
services:
    telegram-bot-sonarr:
        image: woiza/telegram-bot-sonarr
        mem_limit: 128M
        container_name: telegram-bot-sonarr
#        depends_on:
#            - sonarr
        restart: always
        environment:
            - SBOT_TELEGRAM_BOT_TOKEN=1460...:AAHlBW_mabVg...
            - SBOT_BOT_ALLOWED_USERIDS=123,987,-567 # Telegram user ID(s), Group IDs are negative
            - SBOT_BOT_MAX_ITEMS=10 # pagination
            - SBOT_BOT_IGNORE_TAGS=false # true/false; true = bot will not ask for tags (useful with auto-tagging)
            - SBOT_SONARR_PROTOCOL=http # http or https
            - SBOT_SONARR_PORT=8989
            - SBOT_SONARR_HOSTNAME=192.168.2.2 # IP or hostname
            - SBOT_SONARR_BASE_URL= # optional, e.g. /sonarr, depending on sonarr configuration
            - SBOT_SONARR_API_KEY=1010d7...
```
### Commands for Botfather's /setcommands

```
q - searches a series 
library - lists all series - WARNING: can be large
delete - deletes series - WARNING: can be large
clear - deletes all previously sent commands
free - lists the free space of your disks
up - lists upcoming episodes in the next 30 days
rss - performs a RSS sync
system - shows your Sonarr configuration
id - shows your Telegram user ID
```

## Contributing
Feel free to contribute to this Telegram bot by submitting pull requests, reporting issues, or suggesting enhancements. Your contributions are welcome!

## Beer
If you appreciate what we do, consider treating us to a refreshing beverage.

<a href="https://paypal.me/telegramarrbots?country.x=EUR" target="_blank">
  <img src="pp.png?raw=true" alt="q1" title="donate" width="200">
</a>

## License
This Telegram bot is licensed under the [MIT License](https://opensource.org/license/mit/).