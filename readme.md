`mkvdualsub` is a tool for combining two subtitle tracks from `mkv` file into one combined `ass` file to display two subtitle tracks simultaneously (one on screen top, another on screen bottom).

Prerequisites:

* [mkvtoolnix](https://mkvtoolnix.download/downloads.html) installed (to see subtitle track information and extract `srt` files)
* a working internet connection (since subtitles combined by submitting a form to https://pas-bien.net/2srt2ass/ and downloading `ass` file)

## Motivation

I am using VLC v3 on MacOS for watching videos.

When I decided to improve my English language a bit I found some films in `mkv` format that had both Russian and English subtitle tracks.

Unfortunately VLC v3 does not allow enabling several subtitle tracks – to simultaneously have both tracks on screen (VLC v4 beta does have this feature, but I had problems with it).

So I am Russian, watching movie with English language audio, looking both on English and Russian subtitles. Like this:

![example](https://raw.githubusercontent.com/FZambia/mkvdualsub/master/example.png)

## Usage

Suppose you have a video file: `Gravity.Falls.S01.E01.720p.mkv`.

List available subtitle tracks:

```console
mkvdualsub info Gravity.Falls.S01.E01.720p.mkv
```

Output will be like:

```console
Track ID 5: subtitles (SubRip/SRT)
Track ID 6: subtitles (SubRip/SRT)
```

Now you can get `ass` file with combined tracks using:

```console
mkvdualsub join Gravity.Falls.S01.E01.720p.mkv -t 6 -b 5
```

– where `-t` sets top subtitle track number, `-b` sets bottom subtitle track number.

Run video and enjoy combined subtitles (now located inside `Gravity.Falls.S01.E01.720p.mkv.ass` file in the same directory as video).

If you simply want to join first two subtitle tracks in `mkv` run:

```console
mkvdualsub join Gravity.Falls.S01.E01.720p.mkv
```
