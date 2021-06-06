# Sonarr/Radarr Torrent Cleaner

Simple executables to remove torrents and optionally blacklist them if they haven't progressed in a set time period.

## Usage

- Rename the provided `cleaner_config.default.json`, to be `rtcleaner_config.json` or `stcleaner_config.json` and set it up for your preference.
- Run the executable every x minutes using an external program such as cron (Built in scheduling coming soon) (Recommended time 15 minutes)

## Config

``` json
{
    //Time to wait on a torrent that has made no progress before removing it (Time format: https://golang.org/pkg/time/#ParseDuration)
    "WaitTime": "4h",
    // Your Sonarr api key
    "SonarrAPIKey": "xxxxxxxxxxxxxxxx",
    // The address your Sonnarr install can be found at
    "SonarrURL": "http://localhost",
    // The amount of time to wait for a torrent to get past 0% before removing it
    "ZeroPercentTimeout": "1h",
    // Blacklist the torrent in Sonarr so it's not downloaded again (Time format: https://golang.org/pkg/time/#ParseDuration)
    "Blacklist" : true
}
```

## Scheduling

Currently this does not have a built in scheduler, on linux this is easy to do with cron (see below) on windows you have a few options [see here](https://stackoverflow.com/a/132975)

- On Linux, you can simply run: `crontab -e` and add the following to the bottom of the page.

`*/15 * * * *  go run /path/to/TorrentCleaner/stcleaner.go`

`*/15 * * * *  go run /path/to/TorrentCleaner/rtcleaner.go`
- This will run the cleaners every 15 minutes.
- You can run `crontab -l` to check your user crontab, or delete it with `crontab -r`. (This deletes the any cron jobs listed with `-l`)
