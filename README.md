# Sonarr/Radarr Torrent Cleaner

Simple executables to remove torrents and optionally blacklist them if they haven't progressed in a set time period.

## Usage

- Rename the provided `cleaner_config.default.json`, to be `rtcleaner_config.json` or `stcleaner_config.json` and set it up for your preference.
- Run the executable every x minutes using an external program such as cron (Built in scheduling coming soon) (Recommended time 15 minutes)

## Config

``` json
{
"SonarrURL": "http://localhost:8989",
"RadarrURL": "http://localhost:7878",
"SonarrAPIKey": "xxxxxxxxxxxxxxxx",
"RadarrAPIKey": "xxxxxxxxxxxxxxxx",
"WaitTime": "4h",
"ZeroPercentTimeout": "1h",
"Blacklist" : true
}

```
# But for why???
-`SonarrURL|RadarrURL` The URL address of your apps (set to the app default, change as needed for your setup)
-`SonarrAPIKey|RadarrAPIKey` Your api keys, which can be found on the Arr's webUI -> settings -> general
-`WaitTime` Timer for stalled downloads before removing it.
-`ZeroPercentTimeout` Timer for a download to get away from 0% (paused or queued don't count).
-`Blacklist` Set to Blacklist the torrent in app so it's not automatically pulled and downloaded again.
[Time Format](Time format: https://golang.org/pkg/time/#ParseDuration)

## Scheduling

Currently this does not have a built in scheduler, on linux this is easy to do with cron (see below) on windows you have [a few options](https://stackoverflow.com/a/132975)

- On Linux, you can simply run: `crontab -e` and add the following to the bottom of the page.

`*/15 * * * *  go run /path/to/TorrentCleaner/stcleaner.go`

`*/15 * * * *  go run /path/to/TorrentCleaner/rtcleaner.go`
- This will run the cleaners every 15 minutes.
- You can run `crontab -l` to check your user crontab, or delete it with `crontab -r`. (This deletes the any cron jobs listed with `-l`)
