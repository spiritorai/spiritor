package transcribe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/spiritorai/spiritor/utils"
)

// TODO: The api key is required for whisper. It is read in from API_KEY_OPENAI by
// init(). This op should be moved to the scribe package so as to be command specific
// and return a cleaner error.
var apiKey string

func init() {
	apiKey = os.Getenv("API_KEY_OPENAI")
	if apiKey == "" {
		log.Fatal("Missing value for API_KEY_OPENAI: Please set the env var.")
	}
}

const (
	outputTXT  = "txt"
	outputJSON = "json"
	outputSRT  = "srt"
	outputVTT  = "vtt"
)

var supportedOutput = map[string]struct{}{
	outputTXT: {},
}

func OutputAllowed(output string) bool {
	if _, ok := supportedOutput[output]; !ok {
		return false
	}
	return true
}

const (
	whisperMaxBytes        int64 = 25 * 1024 * 1024 // 25mb
	whisperMaxBytesPadding int64 = 100 * 1024       // 100kb
)

func MaxUploadSize() int64 {
	return (whisperMaxBytes - whisperMaxBytesPadding)
}

func Transcribe(ctx context.Context, inputPath string) (Transcript, error) {

	var ts Transcript

	targetFile, err := os.Open(inputPath)
	if err != nil {
		return ts, fmt.Errorf("failed to open file %v: %v", inputPath, err)
	}
	defer targetFile.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filepath.Base(inputPath))
	if err != nil {
		return ts, fmt.Errorf("failed to create form file %v: %v", inputPath, err)
	}

	_, err = io.Copy(part, targetFile)
	if err != nil {
		return ts, fmt.Errorf("failed to write file %v to part: %v", inputPath, err)
	}

	if err := writer.WriteField("model", "whisper-1"); err != nil {
		return ts, fmt.Errorf("failed to write field: model: %v", err)
	}

	if err := writer.WriteField("language", "en"); err != nil {
		return ts, fmt.Errorf("failed to write field: model: %v", err)
	}

	if err := writer.WriteField("response_format", "json"); err != nil {
		return ts, fmt.Errorf("failed to write field: response_format: %v", err)
	}

	/*
		TODO: I want verbose_json with words and/or segments but currently the API is not
		behaving as expected.

		if err := writer.WriteField("response_format", "verbose_json"); err != nil {
			return ts, fmt.Errorf("failed to write field: response_format: %v", err)
		}

		if err := writer.WriteField("timestamp_granularities", "word"); err != nil {
			return ts, fmt.Errorf("failed to write field: timestamp_granularities=word: %v", err)
		}

		if err := writer.WriteField("timestamp_granularities", "segment"); err != nil {
			return ts, fmt.Errorf("failed to write field: timestamp_granularities=segment: %v", err)
		}
	*/

	err = writer.Close()
	if err != nil {
		return ts, fmt.Errorf("failed to close writer: %v", err)
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/audio/transcriptions", body)
	if err != nil {
		return ts, fmt.Errorf("failed create new http request: %v", err)
	}

	req.Header.Add("Content-Type", writer.FormDataContentType())
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %v", apiKey))

	client := &http.Client{}

	// TODO: Timeout the request?

	resp, err := client.Do(req)
	if err != nil {
		return ts, fmt.Errorf("failed to execute http request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errMsg string
		if body, err := io.ReadAll(resp.Body); err == nil {
			errMsg = string(body)
		}
		return ts, fmt.Errorf("request failed with status: %v: %v", resp.StatusCode, errMsg)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return ts, fmt.Errorf("failed to read response body: %v", err)
	}

	if err := json.Unmarshal(respBody, &ts); err != nil {
		return ts, fmt.Errorf("failed to marshal json resp: %v", err)
	}

	//fmt.Printf("TRANSCRIPT:\n%+v", ts)

	return ts, nil
}

type Transcript struct {
	Task     string    `json:"task"`
	Language string    `json:"language"`
	Duration float64   `json:"duration"`
	Text     string    `json:"text"`
	Words    []Word    `json:"words"`
	Segments []Segment `json:"segments"`
}

type Word struct {
	Word  string  `json:"word"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

type Segment struct {
	ID               int     `json:"id"`
	Seek             int     `json:"seek"`
	Start            float64 `json:"start"`
	End              float64 `json:"end"`
	Text             string  `json:"text"`
	Tokens           []int   `json:"tokens"`
	Temperature      float64 `json:"temperature"`
	AvgLogprob       float64 `json:"avg_logprob"`
	CompressionRatio float64 `json:"compression_ratio"`
	NoSpeechProb     float64 `json:"no_speech_prob"`
}

func (ts Transcript) Format(output string) ([]byte, error) {
	switch output {
	case outputTXT:

		sentences, err := utils.SplitSentences(ts.Text)
		if err != nil {
			return nil, fmt.Errorf("failed to split sentences: %w", err)
		}

		return []byte(utils.CombineSentences(sentences, "\n\n")), nil

	default:
		return nil, fmt.Errorf("output format not supported: %v", output)
	}
}
