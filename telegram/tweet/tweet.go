package tweet

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/enescakir/emoji"
	"github.com/g8rswimmer/go-twitter"
	"github.com/thoas/go-funk"
	"github.com/yangrq1018/jerry-bot/telegram"
	tgbotapi "github.com/yangrq1018/telegram-bot-api/v5"
	"golang.org/x/oauth2/clientcredentials"
)

const (
	myID = "1162684522904743938"
)

// a dumb placeholder for Authorizer
// the actual authorization is done by customize http.Client
type authorizeNull struct{}

func (a authorizeNull) Add(_ *http.Request) {
	return
}

func oAuth2Config() *clientcredentials.Config {
	return &clientcredentials.Config{
		ClientID:     os.Getenv("TWITTER_API_KEY"),
		ClientSecret: os.Getenv("TWITTER_API_SECRET"),
		TokenURL:     "https://api.twitter.com/oauth2/token", // server token endpoint
	}
}

func newUserOAuth2() *twitter.User {
	return &twitter.User{
		Authorizer: authorizeNull{},
		Client:     oAuth2Config().Client(context.TODO()),
		Host:       "https://api.twitter.com",
	}
}

func newLookup() *twitter.Tweet {
	return &twitter.Tweet{
		Authorizer: authorizeNull{},
		Client:     oAuth2Config().Client(context.TODO()),
		Host:       "https://api.twitter.com",
	}
}

type tweet struct {
	user *twitter.User
	twl  *twitter.Tweet
}

func (t tweet) ID() tgbotapi.BotCommand {
	return tgbotapi.BotCommand{
		Command:     "nintendo",
		Description: "同步switch截图和录屏",
	}
}

func (t *tweet) Serve(bot *telegram.Bot) error {
	bot.Match(t).Subscribe(t.handle)
	re := regexp.MustCompile(`nintendo(_\w+)?`)
	bot.UpdateEvent.Subscribe(func(b *telegram.Bot, u tgbotapi.Update) error {
		if re.MatchString(u.Message.Command()) {
			return t.handle(b, u)
		}
		return nil
	})
	return nil
}

func (t *tweet) Init() {
	t.user = newUserOAuth2()
	t.twl = newLookup()
}

func (t tweet) Authorize() telegram.Authorizer {
	return telegram.PolicyAllow
}

func (t *tweet) handle(b *telegram.Bot, u tgbotapi.Update) error {
	cmd := u.Message.Command()
	if cmd == "nintendo" {
		// query the last 20 tweets
		tweets, err := getTweets(t.user, 20)
		if err != nil {
			return err
		}
		lines := SearchTimeline(tweets.Tweets, *tweets.Includes)
		msg := tgbotapi.NewMessage(u.Message.Chat.ID, strings.Join(lines, "\n"))
		msg.ParseMode = "HTML"
		_, err = b.Bot().Send(msg)
	} else {
		twID := strings.ReplaceAll(cmd, "nintendo_", "")
		tw, media, err := getTweet(t.twl, twID)
		if err != nil {
			return err
		}
		if tw == nil {
			return fmt.Errorf("this tweet not found in user timeline")
		}
		if media == nil {
			return fmt.Errorf("this tweet has no media content")
		}
		var msg tgbotapi.Chattable
		switch media.Type {
		case "photo":
			imageURl, _ := url.Parse(media.URL)
			// use value (url.URL) to match "case url.URL"
			msg = tgbotapi.NewPhotoUpload(
				u.Message.Chat.ID,
				*imageURl,
			)
		case "video":
			// todo video url is not supported in v2, this is a known issue of Twitter API v2
			if media.URL == "" {
				return fmt.Errorf("media url (video) is empty")
			}
			videoURL, urlErr := url.Parse(media.URL)
			if urlErr != nil {
				return fmt.Errorf("cannot parse media url (video): %v", urlErr)
			}
			msg = tgbotapi.NewVideoUpload(
				u.Message.Chat.ID,
				*videoURL,
			)
		}
		if msg != nil {
			_, _ = b.Bot().Send(msg)
		}
	}
	return nil
}

func Command() telegram.Command {
	return new(tweet)
}

func getMediaOfTweet(tw twitter.TweetObj, include twitter.UserTimelineIncludes) (objs []twitter.MediaObj) {
	for k := range include.Medias {
		if funk.ContainsString(tw.Attachments.MediaKeys, include.Medias[k].Key) {
			objs = append(objs, include.Medias[k])
		}
	}
	return objs
}

func SearchTimeline(tweets []twitter.TweetObj, include twitter.UserTimelineIncludes) []string {
	var lines []string
	for _, tw := range tweets {
		var line string
		if !hasNintendoHashTag(tw) {
			continue
		}

		var (
			media      twitter.MediaObj
			game       = "unknown"
			created, _ = time.Parse(time.RFC3339, tw.CreatedAt)
		)

		for _, hashTag := range tw.Entities.HashTags {
			if hashTag.Tag != "NintendoSwitch" {
				game = hashTag.Tag
				break
			}
		}

		medias := getMediaOfTweet(tw, include)
		if len(medias) == 0 {
			continue
		}

		media = extractFirst(medias)
		link := "/nintendo_" + tw.ID
		line = mediaTypeEmoji(media.Type).String() + fmt.Sprintf("%s %s(%s)\n%s", created.Format("2006-01-02"), media.Type, game, link)
		lines = append(lines, line)
	}
	return lines
}

// Get the tweet with id on the authenticated user timeline
func getTweet(lookup *twitter.Tweet, id string) (*twitter.TweetObj, *twitter.MediaObj, error) {
	twl, err := lookup.Lookup(context.Background(), []string{id}, twitter.TweetFieldOptions{
		TweetFields: []twitter.TweetField{
			twitter.TweetFieldEntities,
			twitter.TweetFieldCreatedAt,
		},
		MediaFields: []twitter.MediaField{ // for media
			twitter.MediaFieldType,
			twitter.MediaFieldURL,
			twitter.MediaFieldMediaKey,
		},
		Expansions: []twitter.Expansion{
			twitter.ExpansionAttachmentsMediaKeys, // for media
		},
	})
	if err != nil {
		return nil, nil, err
	}
	tw, ok := twl[id]
	if !ok {
		return nil, nil, nil
	}
	if tw.Tweet.ID != id {
		return nil, nil, nil
	}
	// video content goes to field AttachmentMedia, not field Media
	var media *twitter.MediaObj
	if len(tw.AttachmentMedia) > 0 {
		media = tw.AttachmentMedia[0]
	}
	return &tw.Tweet, media, nil
}

// Get the most recent count number of tweets of on the authenticated user timeline
func getTweets(user *twitter.User, count int) (*twitter.UserTimeline, error) {
	tweets, err := user.Tweets(context.Background(), myID, twitter.UserTimelineOpts{
		MaxResults: count,
		Excludes: []twitter.Exclude{
			twitter.ExcludeRetweets,
		},
		TweetFields: []twitter.TweetField{ // for media
			twitter.TweetFieldEntities,
			twitter.TweetFieldCreatedAt,
		},
		MediaFields: []twitter.MediaField{ // for media
			twitter.MediaFieldType,
			twitter.MediaFieldURL,
			twitter.MediaFieldMediaKey,
		},
		Expansions: []twitter.Expansion{
			twitter.ExpansionAttachmentsMediaKeys, // for media
		},
	})
	if err != nil {
		return nil, err
	}
	return tweets, err
}

func hasNintendoHashTag(tw twitter.TweetObj) bool {
	for _, ht := range tw.Entities.HashTags {
		if ht.Tag == "NintendoSwitch" {
			return true
		}
	}
	return false
}

func mediaTypeEmoji(mediaType string) emoji.Emoji {
	switch mediaType {
	case "video":
		return emoji.VideoCamera
	case "photo":
		return emoji.Camera
	default:
		return emoji.QuestionMark
	}
}

// assume length of entities greater than zero
func extractFirst(entities []twitter.MediaObj) (me twitter.MediaObj) {
	if entities[0].Type == "video" {
		// video media
		me = entities[0]
	}
	return entities[0]
}
