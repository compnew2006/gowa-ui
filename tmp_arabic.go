package main

import (
	"fmt"
	"regexp"
)

func main() {
	text := "Here is a number ٠١٠٠٧١٨١٧٨١ and another ٢٠١٠٠٧١٨١٧٨١ and +44 7911 123456 and 00 447911"

	intlPhoneRegex := regexp.MustCompile(`(?:^|[^\p{Nd}])((?:\+|00|0|٠٠|٠)[\s\-\.]?[\p{Nd}][\p{Nd}\s\-\.]{5,18}[\p{Nd}]|[\p{Nd}][\p{Nd}\s\-\.]{6,18}[\p{Nd}])(?:[^\p{Nd}]|$)`)

	matches := intlPhoneRegex.FindAllStringSubmatchIndex(text, -1)
	for _, matchIdxs := range matches {
		rawNumber := text[matchIdxs[2]:matchIdxs[3]]
		fmt.Printf("Raw: '%s'\n", rawNumber)
	}
}
