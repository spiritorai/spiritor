package scribe

import (
	"context"
	"fmt"

	"github.com/spiritorai/spiritor/avmedia"
	"github.com/spiritorai/spiritor/transcribe"
)

// Pattern: https://go.dev/play/p/bqisBD1y2hI

type Job struct {
	SourceMedia avmedia.Media
	TargetMedia avmedia.Media
	Transcript  transcribe.Transcript
	Err         error
}

func DownsampleWorker(
	ctx context.Context,
	debug bool,
	workdir string,
	jobs <-chan Job,
	success chan<- Job,
	failed chan<- Job,
) {
	for job := range jobs {

		fmt.Printf("downsampling: %v...\n", job.SourceMedia.GetName())

		targetMedia, err := job.SourceMedia.DownsampleOGG(ctx, debug, avmedia.DownsampleOGGConfig{
			OutputBasePath: workdir,
			SizeCap:        transcribe.MaxUploadSize(),
			Strategy:       avmedia.DownsampleStrategyAutoBest,
		})
		if err != nil {
			// TODO: Handle size cap error type and skip instead of exit
			job.Err = fmt.Errorf("media transform failed: %v", err)
			failed <- job
			continue
		}

		job.TargetMedia = targetMedia
		success <- job
	}
}

func TranscriptionWorker(
	ctx context.Context,
	debug bool,
	workdir string,
	jobs <-chan Job,
	success chan<- Job,
	failed chan<- Job,
) {
	for job := range jobs {

		fmt.Printf("transcribing: %v...\n", job.SourceMedia.GetName())

		transcript, err := transcribe.Transcribe(ctx, job.TargetMedia.GetPath())
		if err != nil {
			job.Err = fmt.Errorf("transcribe failed: %v", err)
			failed <- job
			continue
		}

		job.Transcript = transcript
		success <- job
	}
}
