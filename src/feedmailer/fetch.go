package main

import (
	"fmt"
	"github.com/jpoehls/gophermail"
	rss "github.com/jteeuwen/go-pkg-rss"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"html"
	"log"
	"net/smtp"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

var fetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Fetch feeds",
	Long:  `feedmailer will fetch all feeds listed in the config file.`,
	Run:   fetchRun,
}

var fetchWaiter sync.WaitGroup

type Config struct {
	Feeds []string
}

type Itm struct {
	Date         time.Time
	Key          string
	ChannelKey   string
	Title        string
	FullContent  string
	Links        []rss.Link
	Description  string
	Author       rss.Author
	Categories   []*rss.Category
	Comments     string
	Enclosures   []*rss.Enclosure
	Guid         *string
	Source       *rss.Source
	PubDate      string
	Id           string
	Generator    *rss.Generator
	Contributors []string
	Content      *rss.Content
	Extensions   map[string]map[string][]rss.Extension
}

type Chnl struct {
	Url            string
	Key            string
	Title          string
	Links          []rss.Link
	Description    string
	Language       string
	Copyright      string
	ManagingEditor string
	WebMaster      string
	PubDate        string
	LastBuildDate  string
	Docs           string
	Categories     []*rss.Category
	Generator      rss.Generator
	TTL            int
	Rating         string
	SkipHours      []int
	SkipDays       []int
	Image          rss.Image
	ItemKeys       []string
	Cloud          rss.Cloud
	TextInput      rss.Input
	Extensions     map[string]map[string][]rss.Extension
	Id             string
	Rights         string
	Author         rss.Author
	SubTitle       rss.SubTitle
}

func init() {
	fetchCmd.Flags().Int("rsstimeout", 5, "Timeout (in min) for RSS retrival")
	viper.BindPFlag("rsstimeout", fetchCmd.Flags().Lookup("rsstimeout"))
}

func fetchRun(cmd *cobra.Command, args []string) {
	Fetcher()

	// Provides a way to cancel the feed fetching
	// when it is setup to run forever.
	//
	// sigChan := make(chan os.Signal, 1)
	// signal.Notify(sigChan, os.Interrupt)
	// <-sigChan
}

func Fetcher() {
	var config Config

	if err := viper.Marshal(&config); err != nil {
		fmt.Println(err)
	}

	// Get our bookmarks to remember
	// which items we've already sent.
	bookmarks, err := getBookmarks()
	if err != nil {
		fmt.Println(err)
		return
	}

	// Fetch all the feeds concurrently.
	for _, feed := range config.Feeds {
		fetchWaiter.Add(1)
		go PollFeed(feed)
	}

	fetchWaiter.Wait()
	fmt.Println("Done fetching feeds")

	// Build the model we'll pass to our email templates.
	pruneSentItems(bookmarks)
	pruneEmptyChannels()
	fmt.Println("Done pruning")

	// Build the email.
	model := &EmailModel{
		Channels:    channels,
		Items:       items,
		FetchErrors: fetchErrors,
	}

	plainText, err := renderPlainText(model)
	if err != nil {
		fmt.Println(err)
		return
	}

	htmlText, err := renderHtmlText(model)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Send it.
	//fmt.Println(plainText)
	//fmt.Println(htmlText)
	//fmt.Println(items[0].FullContent)

	msg := &gophermail.Message{}
	msg.SetFrom(viper.GetString("send_from"))
	msg.AddTo(viper.GetString("send_to"))
	msg.Subject = viper.GetString("subject")
	msg.Body = plainText
	msg.HTMLBody = htmlText
	smtpAuth := smtp.PlainAuth("", viper.GetString("smtp_user"), viper.GetString("smtp_pass"), viper.GetString("smtp_server"))
	err = gophermail.SendMail(viper.GetString("smtp_server")+":"+viper.GetString("smtp_port"), smtpAuth, msg)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Sent to " + viper.GetString("send_to"))

	bookmarks = updateBookmarks(bookmarks)
	err = saveBookmarks(bookmarks)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Bookmarks updated")
}

func PollFeed(uri string) {
	defer fetchWaiter.Done()

	timeout := viper.GetInt("RSSTimeout")
	if timeout < 1 {
		timeout = 1
	}
	feed := rss.New(timeout, true, chanHandler, itemHandler)

	for {
		if err := feed.Fetch(uri, nil); err != nil {
			fmt.Fprintf(os.Stderr, "[e] %s: %s\n", uri, err)

			insertFetchError(&fetchError{
				ChannelUrl: uri,
				Error:      err.Error(),
			})
			return
		}

		break

		// Uncomment later if we want to have the fetcher
		// run continuously.
		//
		//fmt.Printf("Sleeping for %d seconds on %s\n", feed.SecondsTillUpdate(), uri)
		//time.Sleep(time.Duration(feed.SecondsTillUpdate() * 1e9))
	}
}

func chanHandler(feed *rss.Feed, newchannels []*rss.Channel) {
	fmt.Printf("%d new channel(s) in %s\n", len(newchannels), feed.Url)
	for _, ch := range newchannels {
		chnl := chnlify(feed.Url, ch)
		insertChannel(&chnl)
	}
}

func itemHandler(feed *rss.Feed, ch *rss.Channel, newitems []*rss.Item) {
	fmt.Printf("%d new item(s) in %s\n", len(newitems), feed.Url)
	for _, item := range newitems {
		itm := itmify(item, ch)
		insertItem(&itm)
	}
}

func itmify(o *rss.Item, ch *rss.Channel) Itm {
	var x Itm
	x.Title = o.Title
	for _, l := range o.Links {
		if l == nil {
			continue
		}
		x.Links = append(x.Links, *l)
	}
	x.ChannelKey = ch.Key()
	x.Description = o.Description
	x.Author = o.Author
	x.Categories = o.Categories
	x.Comments = o.Comments
	x.Enclosures = o.Enclosures
	x.Guid = o.Guid
	x.PubDate = o.PubDate
	x.Id = o.Id
	x.Key = o.Key()
	x.Generator = o.Generator
	x.Contributors = o.Contributors
	x.Content = o.Content
	x.Extensions = o.Extensions
	x.Date, _ = o.ParsedPubDate()

	if o.Content != nil && o.Content.Text != "" {
		x.FullContent = o.Content.Text
	} else {
		x.FullContent = o.Description
	}

	x.FullContent = strings.TrimSpace(x.FullContent)

	// Remove some junk.
	x.FullContent = strings.TrimPrefix(x.FullContent, "<http://purl.org/rss/1.0/modules/content/:encoded>")
	x.FullContent = strings.TrimSuffix(x.FullContent, "</http://purl.org/rss/1.0/modules/content/:encoded>")

	// Check if the content is escaped and unescape it.
	if strings.HasPrefix(x.FullContent, "&lt;") {
		x.FullContent = html.UnescapeString(x.FullContent)
	}

	return x
}

func chnlify(url string, o *rss.Channel) Chnl {
	var x Chnl
	x.Url = url
	x.Key = o.Key()
	x.Title = o.Title
	x.Links = o.Links
	x.Description = o.Description
	x.Language = o.Language
	x.Copyright = o.Copyright
	x.ManagingEditor = o.ManagingEditor
	x.WebMaster = o.WebMaster
	x.PubDate = o.PubDate
	x.LastBuildDate = o.LastBuildDate
	x.Docs = o.Docs
	x.Categories = o.Categories
	x.Generator = o.Generator
	x.TTL = o.TTL
	x.Rating = o.Rating
	x.SkipHours = o.SkipHours
	x.SkipDays = o.SkipDays
	x.Image = o.Image
	x.Cloud = o.Cloud
	x.TextInput = o.TextInput
	x.Extensions = o.Extensions
	x.Id = o.Id
	x.Rights = o.Rights
	x.Author = o.Author
	x.SubTitle = o.SubTitle

	var keys []string
	for _, y := range o.Items {
		keys = append(keys, y.Key())
	}
	x.ItemKeys = keys

	return x
}

func (i Itm) FirstLink() (link rss.Link) {
	if len(i.Links) == 0 {
		return
	}
	return i.Links[0]
}

func (c Chnl) HomePage() string {
	if len(c.Links) == 0 {
		return ""
	}

	url, err := url.Parse(c.Links[0].Href)
	if err != nil {
		log.Println(err)
	}
	return url.Scheme + "://" + url.Host
}
