package gogh

import (
	"bytes"
	"strings"
	"unicode"
)

// Public returns a Go-public golint-aware camel cased word built upon head and parts joined with _
func Public(head string, parts ...string) string {
	return toCamelCase(true, head, parts...)
}

// Private same as public, just Go-private
func Private(head string, parts ...string) string {
	return toCamelCase(false, head, parts...)
}

// Underscored returns underscored case of a word
func Underscored(head string, parts ...string) string {
	if strings.IndexByte(head, '_') >= 0 {
		ps := strings.Split(head, "_")
		ps = append(ps, parts...)
		j := 0
		for _, part := range ps {
			if part != "" {
				ps[j] = Underscored(part)
				j++
			}
		}
		return strings.Join(ps[:j], "_")
	}

	rname := []rune(head)
	upMap := make([]bool, len(rname))
	for i, r := range rname {
		if unicode.IsUpper(r) {
			upMap[i] = true
		}
	}
	var buf bytes.Buffer
	var passRunes int
	var putUnderscoreAfterPass bool
	for i, r := range rname {
		if passRunes > 0 {
			passRunes--
			buf.WriteRune(unicode.ToLower(r))
			if passRunes == 0 && putUnderscoreAfterPass {
				buf.WriteByte('_')
				putUnderscoreAfterPass = false
			}
			continue
		}
		buf.WriteRune(unicode.ToLower(r))
		if upMapMatch(upMap[i:], false, true) && r != '_' {
			buf.WriteByte('_')
		}
		var j int
		for j = i; j < len(upMap) && upMap[j]; j++ {
		}
		if _, ok := commonInitialisms[string(rname[i:j])]; ok {
			switch {
			case j == len(upMap):
				passRunes = j - i
			case rname[j] == 's' && j == len(upMap)-1:
				passRunes = j - i + 1
				continue
			case rname[j] == '_':
				passRunes = j - i
				continue
			case rname[j] == 's' && upMapMatch(upMap[j+1:], true):
				passRunes = j - i
				putUnderscoreAfterPass = true
				continue
			case rname[j] == 's' && j < len(upMap)-1 && rname[j+1] == '_':
				passRunes = j - i
				continue
			}
		}
		if upMapMatch(upMap[i:], true, true, false) {
			buf.WriteByte('_')
		}
	}
	return buf.String()
}

// Striked returns striked case of a word, this is same as Underscored, just with - instead of _
func Striked(head string, parts ...string) string {
	return strings.ReplaceAll(Underscored(head, parts...), "_", "-")
}

// Proto returns Go-public camel cased word matching protoc-gen-go
func Proto(head string, parts ...string) string {
	return toProtoCamelCase(true, head, parts...)
}

func toCamelCase(public bool, head string, parts ...string) string {
	var buf strings.Builder
	split := strings.Split(head, "_")
	for i, item := range append(split, parts...) {
		if public || i > 0 {
			uppered := strings.ToUpper(item)
			var candidate string
			if strings.HasSuffix(item, "s") {
				candidate = uppered[:len(uppered)-1]
			}
			var symptom bool
			var exactMatch bool
			if _, ok := commonInitialisms[uppered]; ok {
				symptom = true
				exactMatch = true
			} else if _, ok := commonInitialisms[candidate]; ok {
				symptom = true
			}
			if symptom {
				if exactMatch {
					item = uppered
				} else {
					item = candidate + "s"
				}
			}
			buf.WriteString(strings.Title(item))
		} else {
			buf.WriteString(item)
		}
	}
	return escapeReserveds(buf.String())
}

// toProtoCamelCase camel case just like protoc-gen-go does
func toProtoCamelCase(public bool, head string, parts ...string) string {
	var buf strings.Builder
	split := strings.Split(head, "_")
	for i, item := range append(split, parts...) {
		if public || i > 0 {
			buf.WriteString(strings.Title(item))
		} else {
			buf.WriteString(item)
		}
	}
	return escapeReserveds(buf.String())
}

func upMapMatch(upMap []bool, pattern ...bool) bool {
	if len(upMap) < len(pattern) {
		return false
	}
	for i, v := range pattern {
		if v != upMap[i] {
			return false
		}
	}
	return true
}

func escapeReserveds(res string) string {
	switch res {
	case `break`, `case`, `chan`, `const`, `continue`, `default`, `defer`,
		`else`, `fallthrough`, `for`, `func`, `go`, `goto`, `if`, `import`,
		`interface`, `map`, `package`, `range`, `return`, `select`, `struct`,
		`switch`, `type`, `var`:
		return res + "Escaped"
	default:
		return res
	}
}
