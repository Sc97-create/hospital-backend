package utils

import (
	"crypto/rand"
	"hospital-backend/pkg/constants"
	"math/big"
	"net/mail"
	"regexp"
	"strings"
	"time"
	"unicode"
)

var namePattern = regexp.MustCompile(`^[\p{L}][\p{L} .'-]*$`)

var dateLayouts = []string{
	"2006-01-02",
	"02-01-2006",
	"02/01/2006",
	"2006/01/02",
	"02 Jan 2006",
	time.RFC3339,
}

func IsValidName(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" && len(value) <= 100 && namePattern.MatchString(value)
}

func IsValidEmail(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	addr, err := mail.ParseAddress(value)
	if err != nil {
		return false
	}
	return strings.EqualFold(addr.Address, value)
}

func NormalizeMobile(value string) (string, bool) {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, " ", "")
	value = strings.ReplaceAll(value, "-", "")
	if strings.HasPrefix(value, "+") {
		value = value[1:]
	}
	if len(value) < 10 || len(value) > 15 {
		return "", false
	}
	for _, r := range value {
		if !unicode.IsDigit(r) {
			return "", false
		}
	}
	return value, true
}

func ParseDate(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}
	for _, layout := range dateLayouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func CreateTempPassword(firstName string, dateOfBirth string) string {
	name := strings.ReplaceAll(strings.TrimSpace(firstName), " ", "")
	return name + compactDOB(dateOfBirth) + randomSpecialChar()
}

func compactDOB(value string) string {
	if parsed, ok := ParseDate(value); ok {
		return parsed.Format("20060102")
	}
	var digits strings.Builder
	for _, r := range value {
		if unicode.IsDigit(r) {
			digits.WriteRune(r)
		}
	}
	return digits.String()
}

func randomSpecialChar() string {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(constants.SpecialChars))))
	if err != nil {
		return "!"
	}
	return string(constants.SpecialChars[n.Int64()])
}
