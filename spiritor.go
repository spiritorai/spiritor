package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/spiritorai/spiritor/avmedia"
	"github.com/spiritorai/spiritor/scribe"
	"github.com/spiritorai/spiritor/transcribe"
)

/*
	TODO:

	* Modify ffmpeg to limit threads and test it out. (https://streaminglearningcenter.com/blogs/ffmpeg-command-threads-how-it-affects-quality-and-performance.html)
	* Worker context propagation and timeouts/cancellations.
	* Test worker errors
	* Re-organize commands into cli/dir structure
	* Implement pretty console output with debug option
		* I should use the new log/slog library for standard go as a base (based on zap!)
		* The library allows you to write your own custom slog handler: https://github.com/golang/example/tree/master/slog-handler-guide
		* I should be able to find a custom slog handler optimized for pretty console output!
		* This custom handler is great for local dev and would be great for debug mode as well: https://dusted.codes/creating-a-pretty-console-logger-using-gos-slog-package
		* A great exercise would be to see if I can get the color/formatting of the ffmpeg logs to come through!
		* This lists all slog libraries, they even have clickhouse forwarding, adapters from zerolog, integration with CHI!
		* Awesome zero-deps dev handler: https://github.com/golang-cz/devslog
	* Support for other outputs, eg: (verbose) json, srt, vtt
	* Implement dependencies test command and/or abort main process if deps fail
		* sys write access
		* ffmpeg
	* Media path validation: Ensure that all paths are absolute and/or cannot be broken and work across multiple OS
	* Make api key configurable
*/

const (
	downsampleWorkerCount    int = 4
	transcriptionWorkerCount int = 6
)

type Context struct {
	Debug bool
}

type ScribeCmd struct {
	Force   bool     `help:"Force overwrite existing transcripts." short:"f" default:false`
	Outputs []string `name:"output" help:"List of output formats." short:"o" default:"txt"`
	Files   []string `arg name:"file" help:"Target file path(s)." type:"path"`
}

func (cmd *ScribeCmd) Run(ctx *Context) error {

	fmt.Printf("\nSpiritor AI: Scribe\n\n")

	if ctx.Debug {
		fmt.Printf("params: force=%v, outputs=%v, files=%v\n", cmd.Force, cmd.Outputs, cmd.Files)
	}

	// Quit now if any unsupported output formats have been given
	for _, output := range cmd.Outputs {
		if !transcribe.OutputAllowed(output) {
			return fmt.Errorf("unsupported output: %v", output)
		}
	}

	workdir, err := os.MkdirTemp("", "spiritor")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %v", err)
	}
	// TODO: For some reason this does not get killed when I force exit
	defer os.RemoveAll(workdir)

	// Parse the initial file paths and extract all files available for processing
	var files []avmedia.Media
	for _, fpath := range cmd.Files {

		if fext := strings.ToLower(strings.TrimLeft(filepath.Ext(fpath), ".")); !avmedia.ExtAllowedDownsampleOGG(fext) {
			if ctx.Debug {
				fmt.Printf("skipped: %v: unsupported ext: %v\n", fpath, fext)
			}
			continue
		}

		sourceMedia, err := avmedia.NewMedia(ctx.Debug, fpath)
		if err != nil {
			return fmt.Errorf("new media wrapper failed: %v", err)
		}

		// Test for outputs now so we can skip early if they already exist. If force
		// flag has been set or at least one specified output does not exist then
		// do not skip the file.
		if !cmd.Force {
			if len(probeOutputs(sourceMedia, cmd.Outputs)) == len(cmd.Outputs) {
				fmt.Printf("skipped: %v: outputs already exist\n", fpath)
				continue
			}
		}

		fmt.Printf("processing: %v\n", fpath)

		files = append(files, sourceMedia)
	}

	// set up job and worker channels
	downsampleJobs := make(chan scribe.Job)
	defer close(downsampleJobs)

	trancriptionJobs := make(chan scribe.Job)
	defer close(trancriptionJobs)

	jobResults := make(chan scribe.Job, len(files))
	defer close(jobResults)

	for w := 1; w <= downsampleWorkerCount; w++ {
		go scribe.DownsampleWorker(context.TODO(), ctx.Debug, workdir, downsampleJobs, trancriptionJobs, jobResults)
	}

	for w := 1; w <= transcriptionWorkerCount; w++ {
		go scribe.TranscriptionWorker(context.TODO(), ctx.Debug, workdir, trancriptionJobs, jobResults, jobResults)
	}

	// kick off the workers
	for _, sourceMedia := range files {
		downsampleJobs <- scribe.Job{
			SourceMedia: sourceMedia,
		}
	}

	// process the results
	for i := 1; i <= len(files); i++ {
		job := <-jobResults
		if job.Err != nil {
			fmt.Printf("failed: %v: %v\n", job.SourceMedia.GetName(), job.Err)
			continue
		}

		for _, output := range cmd.Outputs {
			body, err := job.Transcript.Format(output)
			if err != nil {
				fmt.Printf("failed: %v: transcript format error: %v\n", job.SourceMedia.GetName(), err)
				continue
			}

			outputPath := buildOutputPath(job.SourceMedia, output)
			if err := os.WriteFile(outputPath, body, 0666); err != nil {
				fmt.Printf("failed: %v: file write error: %v\n", job.SourceMedia.GetName(), err)
				continue
			}

			fmt.Printf("succeeded: %v\n", outputPath)
		}

	}

	fmt.Printf("\nAh, the sweet smell of success!\n")
	return nil
}

var cli struct {
	Debug  bool      `help:"Enable debug mode."`
	Scribe ScribeCmd `cmd help:"Generates transcripts for a file."`
}

func main() {
	ctx := kong.Parse(&cli)
	// Call the Run() method of the selected parsed command.
	err := ctx.Run(&Context{Debug: cli.Debug})
	ctx.FatalIfErrorf(err)
}

// probeOutputs will test the full path of each of the passed output formats for an
// existing file and return a new list of formats that already exist.
func probeOutputs(media avmedia.Media, outputs []string) []string {
	exists := []string{}
	for _, output := range outputs {
		if _, err := os.Stat(buildOutputPath(media, output)); err == nil {
			exists = append(exists, output)
		}
	}
	return exists
}

// buildOutputPath will construct and return the full path of the output file based on the
// output format param that is passed. This can be called before or after the output
// file is created.
func buildOutputPath(media avmedia.Media, output string) string {
	return fmt.Sprintf("%v.%v", media.GetPath(), output)
}
