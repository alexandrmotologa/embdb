package bm25

import (
	"strings"
	"unicode"
)

var defaultStopWords = map[string]bool{
	"a": true, "about": true, "above": true, "after": true, "again": true,
	"against": true, "all": true, "am": true, "an": true, "and": true,
	"any": true, "are": true, "as": true, "at": true, "be": true,
	"because": true, "been": true, "before": true, "being": true, "below": true,
	"between": true, "both": true, "but": true, "by": true, "can": true,
	"did": true, "do": true, "does": true, "doing": true, "don": true,
	"down": true, "during": true, "each": true, "few": true, "for": true,
	"from": true, "further": true, "had": true, "has": true, "have": true,
	"having": true, "he": true, "her": true, "here": true, "hers": true,
	"herself": true, "him": true, "himself": true, "his": true, "how": true,
	"i": true, "if": true, "in": true, "into": true, "is": true,
	"it": true, "its": true, "itself": true, "just": true, "me": true,
	"more": true, "most": true, "my": true, "myself": true, "no": true,
	"nor": true, "not": true, "now": true, "of": true, "off": true,
	"on": true, "once": true, "only": true, "or": true, "other": true,
	"our": true, "ours": true, "ourselves": true, "out": true, "over": true,
	"own": true, "s": true, "same": true, "she": true, "should": true,
	"so": true, "some": true, "such": true, "t": true, "than": true,
	"that": true, "the": true, "their": true, "theirs": true, "them": true,
	"themselves": true, "then": true, "there": true, "these": true, "they": true,
	"this": true, "those": true, "through": true, "to": true, "too": true,
	"under": true, "until": true, "up": true, "very": true, "was": true,
	"we": true, "were": true, "what": true, "when": true, "where": true,
	"which": true, "while": true, "who": true, "whom": true, "why": true,
	"will": true, "with": true, "you": true, "your": true, "yours": true,
	"yourself": true, "yourselves": true,
}

// Tokenize processes raw text into normalized, filtered, and stemmed tokens.
func Tokenize(text string) []string {
	var tokens []string
	var current strings.Builder

	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			current.WriteRune(unicode.ToLower(r))
		} else {
			if current.Len() > 0 {
				token := current.String()
				current.Reset()
				if !defaultStopWords[token] && len(token) > 1 {
					tokens = append(tokens, stem(token))
				}
			}
		}
	}

	if current.Len() > 0 {
		token := current.String()
		if !defaultStopWords[token] && len(token) > 1 {
			tokens = append(tokens, stem(token))
		}
	}

	return tokens
}

// stem performs a lightweight English suffix stemmer.
func stem(word string) string {
	if len(word) <= 3 {
		return word
	}

	// Step 1: Plurals and suffixes
	if strings.HasSuffix(word, "sses") {
		return word[:len(word)-2]
	}
	if strings.HasSuffix(word, "ies") {
		return word[:len(word)-2]
	}
	if strings.HasSuffix(word, "ss") {
		return word
	}
	if strings.HasSuffix(word, "s") {
		return word[:len(word)-1]
	}

	// Step 2: Gerunds and past tense
	if strings.HasSuffix(word, "ing") && len(word) > 4 {
		word = word[:len(word)-3]
		// Handle double consonant (e.g. runn -> run, swimm -> swim)
		if len(word) >= 3 && word[len(word)-1] == word[len(word)-2] {
			last := word[len(word)-1]
			if last == 'n' || last == 'p' || last == 't' || last == 'm' || last == 'g' || last == 'b' {
				word = word[:len(word)-1]
			}
		}
		return word
	}
	if strings.HasSuffix(word, "ed") && len(word) > 4 {
		return word[:len(word)-2]
	}
	if strings.HasSuffix(word, "ly") && len(word) > 4 {
		return word[:len(word)-2]
	}
	if strings.HasSuffix(word, "ment") && len(word) > 6 {
		return word[:len(word)-4]
	}
	if strings.HasSuffix(word, "tion") && len(word) > 6 {
		return word[:len(word)-4]
	}

	return word
}
