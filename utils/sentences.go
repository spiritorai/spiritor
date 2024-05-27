package utils

import (
	"fmt"
	"strings"

	"github.com/neurosnap/sentences"
	"github.com/spiritorai/spiritor/neurosnaptrainingdata"
)

// SplitSentences will use the neurosnap/sentences tokenizer to split the content up into
// a slice of strings. Each string is a sentence, based on existing punctuation. This util
// method is hardcoded to english at the moment but the sentence splitter does support
// multiple languages which can be enabled later.
func SplitSentences(content string) ([]string, error) {

	var final []string

	tdata, err := neurosnaptrainingdata.FS.ReadFile("english.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read training data: %w", err)
	}

	training, _ := sentences.LoadTraining(tdata)
	if err != nil {
		return nil, fmt.Errorf("failed to load training data: %w", err)
	}

	// create the default sentence tokenizer
	tokenizer := sentences.NewSentenceTokenizer(training)
	sentences := tokenizer.Tokenize(content)

	for _, s := range sentences {
		final = append(final, strings.TrimSpace(s.Text))
	}

	return final, nil
}

// CombineSentences will take the output from the SplitSentences func and
// combine it back into a single string using the passed delimiter. This may
// be used as a preparation step for writing the sentences all back into a
// single output file.
func CombineSentences(sentences []string, delimeter string) string {
	return strings.Join(sentences[:], delimeter)
}
