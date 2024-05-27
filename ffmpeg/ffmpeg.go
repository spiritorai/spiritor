package ffmpeg

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func execCmd(ctx context.Context, app string, args []string) (string, error) {
	if app == "" {
		return "", fmt.Errorf("exec error: app cannot be empty")
	}

	cmd := exec.Command(app, args...)
	cmd.Env = os.Environ()

	// Note: This enables interactive term proxying!!
	// When this is disabled then interactive prompts trigger
	// an automatic command error.
	// cmd.Stdin = os.Stdin

	//var stdout bytes.Buffer
	//cmd.Stdout = &stdout //>> stderr.String()
	//cmd.Stdout = os.Stderr

	//var stderr bytes.Buffer
	// cmd.Stderr = &stderr //>> stderr.String()
	//cmd.Stderr = os.Stderr

	/*if err := cmd.Run(); err != nil {
		// TODO: Include detailed stderr output
		return fmt.Errorf("cmd run error: %v", err)
	}*/

	// TODO: Execute command async and listen for ctx canceled or timeout and kill op

	// TODO: I would prefer to have outputs flow to console in realtime when debugging
	// is enabled but I would have to implement my own pipeline for that so for now this
	// will have to hack it.

	// Note: Currently this uses CombinedOutput because ffmpeg seems to use the stderr
	// channel explicitely even for successful ops, so this method is just easier. I do
	// not know if the behavior will change on different os.
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), err
	}

	// TODO: only when ctx debug = true
	//fmt.Printf("Exec Output: %v\n", strings.TrimSpace(string(output)))

	return strings.TrimSpace(string(output)), nil
}

// DownsampleOpus will convert the source file to the target file using the libopus
// codec at the target bitrate. The target file path must have an ogg extension and
// must not exist, or else an error will be thrown. The input file must also be
// supported by the underlying ffmpeg operation, or an error will be thrown. Bitrate
// calculations are the responsibility of the caller, unintentional upsample may occur.
func DownsampleOpus(sourceFilePath, targetFilePath, targetBitrate string) error {

	// From: https://community.openai.com/t/whisper-api-increase-file-limit-25-mb/566754
	// ffmpeg -i audio.mp3 -vn -map_metadata -1 -ac 1 -c:a libopus -b:a 12k -application voip audio.ogg

	// My ogg/opus bitrate targets:
	// 96k:
	//   * example_zoom.mp3: No audible loss (37.9mb > 31.4)
	//   * example_zoom.wav: No audible loss (7.9mb > 3mb)
	// 48k: (high)
	//   * example_zoom.mp3: No audible loss (37.9mb > 16.1mb)
	//   * example_zoom.wav: No audible loss (7.9mb > 1.4mb)
	//   * example_signal.aac: No audible loss (1.7mb > 2.5mb)
	// 24k: (medium)
	//   * example_zoom.mp3: Minor audible loss (37.9mb > 8mb)
	//   * example_zoom.wav: No audible loss (7.9mb > 708.9kb)
	//   * example_signal.aac: No audible loss (1.7mb > 1.3mb)
	// 12k: (low)
	//   * example_zoom.mp3: Medium audible loss (37.9mb > 4mb)
	//   * example_zoom.wav: Minor audible loss (7.9mb > 360.4kb)
	//   * example_signal.aac: Minor audible loss (1.7mb > 624.1kb)

	app := "ffmpeg"
	args := []string{
		"-i",
		sourceFilePath,
		"-vn",
		"-map_metadata",
		"-1",
		"-ac",
		"1",
		"-c:a",
		"libopus",
		"-b:a",
		targetBitrate,
		"-application",
		"voip",
		"-threads",
		"4",
		targetFilePath,
	}

	output, err := execCmd(context.TODO(), app, args)
	if err != nil {
		return fmt.Errorf("downsample opus error: %v: %v", err, output)
	}

	return nil
}

func ProbeBitrate(filePath string) (int, error) {

	// From https://stackoverflow.com/questions/47087802/ffmpeg-how-to-convert-audio-to-aac-but-keep-bit-rate-at-what-the-old-file-used
	// #!/usr/bin/env bash
	// AUDIO_BITRATE=`ffprobe -v error -select_streams a:0  -show_entries stream=bit_rate -of default=noprint_wrappers=1:nokey=1 $1`
	// if [[ $AUDIO_BITRATE < 128000 ]]; then
	//   ffmpeg -i $1 -acodec aac -ab ${AUDIO_BITRATE}k -vcodec copy new-$1
	// else
	//   ffmpeg -i $1 -acodec aac -vcodec copy new-$1
	// fi

	app := "ffprobe"
	args := []string{
		"-v",
		"error",
		"-select_streams",
		"a:0",
		"-show_entries",
		"stream=bit_rate",
		"-of",
		"default=noprint_wrappers=1:nokey=1",
		filePath,
	}

	var bitrate int

	output, err := execCmd(context.TODO(), app, args)
	if err != nil {
		return bitrate, fmt.Errorf("probe error: %v: %v", err, output)
	}

	// If the output is "N/A" then return the default bitrate of 0 without error.
	if output == "N/A" {
		return bitrate, nil
	}

	// Only return an error if ffprobe outputs something unexpected.
	bitrate, err = strconv.Atoi(output)
	if err != nil {
		return bitrate, fmt.Errorf("could not covert [%v] to int: %v", output, err)
	}

	return bitrate, nil
}

func ProbeDuration(filePath string) (time.Duration, error) {

	app := "ffprobe"
	args := []string{
		"-v",
		"error",
		"-select_streams",
		"a:0",
		"-show_entries",
		"stream=duration",
		"-of",
		"default=noprint_wrappers=1:nokey=1",
		filePath,
	}

	var duration time.Duration

	output, err := execCmd(context.TODO(), app, args)
	if err != nil {
		return duration, fmt.Errorf("probe error: %v: %v", err, output)
	}

	// Only return an error if ffprobe outputs something unexpected.
	duration, err = time.ParseDuration(fmt.Sprintf("%vs", output))
	if err != nil {
		return duration, fmt.Errorf("could not covert [%v]s to duration: %v", output, err)
	}

	return duration, nil
}

func ProbeDump(filePath string) (string, error) {

	app := "ffprobe"
	args := []string{
		filePath,
	}

	output, err := execCmd(context.TODO(), app, args)
	if err != nil {
		return output, fmt.Errorf("probe error: %v: %v", err, output)
	}

	return output, nil
}
