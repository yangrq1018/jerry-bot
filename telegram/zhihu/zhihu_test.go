package zhihu

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHot(t *testing.T) {
	data, err := Hot()
	assert.NoError(t, err)
	for _, item := range data {
		t.Logf("%s, answer: %d", item.URL, len(item.Answer))
		//for _, ans := range item.Answer {
		//	t.Logf("%s", ans.Prefix(100))
		//	break
		//}
	}
}

func TestQuestion_PrintAnswer(t *testing.T) {
	data, err := Hot()
	assert.NoError(t, err)
	for _, item := range data {
		if len(item.Answer) > 0 {
			fmt.Println(item.PrintAnswer(0))
			break
		}
	}
}

func TestQuestion_QID(t *testing.T) {
	data, err := getHotQuestions()
	assert.NoError(t, err)
	for i, q := range data {
		fmt.Printf("%d %s\n", i, q.QID())
	}
}
