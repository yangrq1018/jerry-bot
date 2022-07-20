package util

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// Min returns the smaller one of x and y
func Min[K float64 | int](x, y K) K {
	if x > y {
		return y
	}
	return x
}

func MustAtoi(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
	return i
}

func StringPtr(s string) *string {
	return &s
}

func PrintIndentedJSON(o interface{}) {
	fmt.Println(IndentedJSON(o))
}

func IndentedJSON(o interface{}) string {
	b, _ := json.MarshalIndent(o, "", "  ")
	return string(b)
}

func DateEqual(t1 time.Time, t2 time.Time) bool {
	y1, m1, d1 := t1.Date()
	y2, m2, d2 := t2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}
