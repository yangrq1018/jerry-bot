package gwy

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHolidays(t *testing.T) {
	names, dates, workdays, err := Holidays(2021)
	assert.NoError(t, err)
	assert.Greater(t, len(names), 0)
	assert.Greater(t, len(dates), 0)
	assert.Greater(t, len(workdays), 0)
}
