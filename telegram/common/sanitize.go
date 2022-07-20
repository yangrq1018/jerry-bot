package common

import "github.com/microcosm-cc/bluemonday"

func PolicySanitizer(elements ...string) *bluemonday.Policy {
	p := bluemonday.NewPolicy()
	p.AllowElements(elements...)
	// allow link with href attribute
	p.AllowAttrs("href").OnElements("a")
	return p
}
