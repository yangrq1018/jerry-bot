package tweet

import (
	"context"
	"fmt"
	"github.com/g8rswimmer/go-twitter"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"time"
)

type authorizeBearer struct {
	Token string
}

func (a authorizeBearer) Add(req *http.Request) {
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", a.Token))
}

func newUserBearerToken() *twitter.User {
	return &twitter.User{
		Authorizer: authorizeBearer{
			Token: os.Getenv("TWITTER_BEARER_TOKEN"),
		},
		Client: http.DefaultClient,
		Host:   "https://api.twitter.com",
	}
}

func newTweet() *twitter.Tweet {
	return &twitter.Tweet{
		Authorizer: authorizeBearer{
			Token: os.Getenv("TWITTER_BEARER_TOKEN"),
		},
		Client: http.DefaultClient,
		Host:   "https://api.twitter.com",
	}
}

func Username2Id(username string) (string, error) {
	user := newUserBearerToken()
	info, err := user.LookupUsername(context.Background(), []string{username}, twitter.UserFieldOptions{})
	if err != nil {
		return "", err
	}
	var id string
	for k := range info {
		if info[k].User.UserName == username {
			id = info[k].User.ID
		}
	}
	return id, nil
}

func UserId2UserInfo(id string) (*twitter.UserObj, error) {
	user := newUserBearerToken()
	info, err := user.Lookup(context.Background(), []string{id}, twitter.UserFieldOptions{})
	if err != nil {
		return nil, err
	}
	u := info[id].User
	return &u, nil
}

func RepliesOfTweet(conversation string, opts twitter.TweetRecentSearchOptions) ([]twitter.TweetObj, error) {
	result, err := newTweet().RecentSearch(context.Background(), "conversation_id:"+conversation, opts, twitter.TweetFieldOptions{
		TweetFields: []twitter.TweetField{
			twitter.TweetFieldCreatedAt,
			twitter.TweetFieldAuthorID,
			twitter.TweetFieldConversationID,
		},
	})
	if err != nil {
		return nil, err
	}
	var replies []twitter.TweetObj
	for k := range result.LookUps {
		replies = append(replies, result.LookUps[k].Tweet)
	}
	return replies, nil
}

func RepliesOfTweetPaginated(conversation string, opts twitter.TweetRecentSearchOptions) ([]twitter.TweetObj, error) {
	var nextToken string
	var tweets []twitter.TweetObj
	for i := 0; nextToken != "" || i == 0; i++ {
		if nextToken != "" {
			opts.NextToken = nextToken
		}
		tws, err := RepliesOfTweet(conversation, opts)
		if err != nil {
			return nil, err
		}
		tweets = append(tweets, tws...)
	}
	return tweets, nil
}

func UserTimeline(id string, start, end time.Time, maxResult int) ([]twitter.TweetObj, error) {
	user := newUserBearerToken()
	tweetOpts := twitter.UserTimelineOpts{
		TweetFields: []twitter.TweetField{
			twitter.TweetFieldCreatedAt,
			twitter.TweetFieldConversationID,
			twitter.TweetFieldEntities,
		},
		Excludes: []twitter.Exclude{
			twitter.ExcludeRetweets,
			twitter.ExcludeReplies,
		},
		MediaFields: []twitter.MediaField{ // for media
			twitter.MediaFieldType,
			twitter.MediaFieldURL,
			twitter.MediaFieldMediaKey,
		},
		Expansions: []twitter.Expansion{
			twitter.ExpansionAttachmentsMediaKeys, // for media
		},
		MaxResults: maxResult,
	}
	if !start.IsZero() {
		tweetOpts.StartTime = start
	}
	if !end.IsZero() {
		tweetOpts.EndTime = end
	}
	var nextToken string
	var tweets []twitter.TweetObj
	for i := 0; nextToken != "" || i == 0; i++ {
		if nextToken != "" {
			tweetOpts.PaginationToken = nextToken
		}
		userTweets, err := user.Tweets(context.Background(), id, tweetOpts)
		if err != nil {
			return nil, err
		}
		nextToken = userTweets.Meta.NextToken
		log.Infof("page %d, token %q, result %d", i+1, nextToken, len(userTweets.Tweets))
		tweets = append(tweets, userTweets.Tweets...)
	}
	return tweets, nil
}
