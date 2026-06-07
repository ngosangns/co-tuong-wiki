package httpapi

import "net/url"

func urlEncode(value string) string {
	return url.QueryEscape(value)
}
