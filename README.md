# Feedmailer

Feedmailer fetches a list of RSS feeds and sends you an email with
all of the new items since the last time you ran it.

Run it daily or weekly to receive a digest email of all your RSS updates.

I created this to scratch a personal itch so it is probably rough around
the edges. If this is something useful to you, let me know!

## Configuration

Feedmailer is configured via a simple config file. By default it expects this to be either at `~/.feedmailer/config.yml`. You can provide your own location with `--config`.

An example file is:

```
subject: "Go Feeds Powered by Feedmailer"
send_to: you@yourdomain.com

send_from: you@yourdomain.com
smtp_server: smtp.gmail.com
smtp_port: 587
smtp_user: you@gmail.com
smtp_pass: yourpassword

feeds:
    - "http://spf13.com/index.xml"
    - "http://dave.cheney.net/feed"
    - "http://www.goinggo.net/feeds/posts/default"
    - "http://blog.labix.org/feed"
    - "http://blog.golang.org/feed.atom"
```

Feedmailer remembers which RSS items it has already sent you by storing
stuff in a `bookmarks.json` file. By default this is stored in the `~/.feedmailer` directory but you can change that by setting the `data_dir` setting in the config file.

## Building

Use the [gb build tool](https://getgb.io).

Run `./check.sh` which will build and lint.

## Thanks

Thanks to https://github.com/spf13/dagobah for the feed fetching logic.

And many, many thanks to all the Go packages I'm using that make this so easy!