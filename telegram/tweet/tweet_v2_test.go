package tweet

import (
	"encoding/csv"
	"io"
	"os"
	"testing"
	"time"

	"github.com/g8rswimmer/go-twitter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var nullTime = NewUTC(1, 1, 1)

func NewUTC(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func TestGetUserTimeline(t *testing.T) {
	tweets, err := UserTimeline(
		myID,
		nullTime,
		nullTime,
		100)
	assert.NoError(t, err)
	assert.Greater(t, len(tweets), 0)
	firstTweet := tweets[0]
	lastTweet := tweets[len(tweets)-1]
	t.Log(firstTweet.CreatedAt, firstTweet.Text, firstTweet.ConversationID)
	t.Log(lastTweet.CreatedAt, lastTweet.Text)
}

func TestRepliesOfTweetPaginated(t *testing.T) {
	conversation := "1504835056912805889"
	opts := twitter.TweetRecentSearchOptions{}
	tweets, err := RepliesOfTweetPaginated(conversation, opts)
	require.NoError(t, err)
	assert.Greater(t, len(tweets), 0)
	for i := range tweets {
		info, _ := UserId2UserInfo(tweets[i].AuthorID)
		t.Log(tweets[i].CreatedAt, info.Name, tweets[i].Text)
	}
}

func csvHandle(filename string) (*csv.Writer, io.Closer, error) {
	f, err := os.Create(filename)
	if err != nil {
		return nil, nil, err
	}
	w := csv.NewWriter(f)
	return w, f, nil
}

func TestGetRepliesOfPosts(t *testing.T) {
	var allReplies, tweets []twitter.TweetObj
	replWriter, f1, err := csvHandle("replies.csv")
	require.NoError(t, err)
	postWriter, f2, err := csvHandle("posts.csv")
	require.NoError(t, err)

	id, err := Username2Id("POTUS")
	require.NoError(t, err)
	start, end := NewUTC(2022, 3, 10), NewUTC(2022, 3, 20)
	tweets, err = UserTimeline(
		id,
		start, end,
		100)
	assert.NoError(t, err)
	opts := twitter.TweetRecentSearchOptions{}
	for i := range tweets {
		replies, err := RepliesOfTweetPaginated(tweets[i].ConversationID, opts)
		require.NoError(t, err)
		if len(replies) == 0 {
			t.Logf("no replies for %s", tweets[i].ConversationID)
			continue
		}
		t.Logf("conversation %q: replies:", tweets[i].ConversationID)
		//for j := range replies {
		//	t.Log(replies[j].CreatedAt, replies[j].Text)
		//}
		allReplies = append(allReplies, replies...)
	}
	saveTweets(tweets, postWriter)
	saveTweets(allReplies, replWriter)

	_ = f1.Close()
	_ = f2.Close()
}

func saveTweets(tweets []twitter.TweetObj, w *csv.Writer) {
	_ = w.Write([]string{
		"id",
		"created_at",
		"text",
		"conversation_id",
		"author_id",
	})
	for _, tw := range tweets {
		_ = w.Write([]string{
			tw.ID,
			tw.CreatedAt,
			tw.Text,
			tw.ConversationID,
			tw.AuthorID,
		})
	}
}

func TestGetUserIdByUsername(t *testing.T) {
	userId, err := Username2Id("UnitedHealthGrp")
	assert.NoError(t, err)
	assert.Equal(t, "917104380", userId)
}
