package avmedia

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/spiritorai/spiritor/ffmpeg"
)

const (
	extMP3  = "mp3"
	extAAC  = "aac"
	extWAV  = "wav"
	extOGG  = "ogg"
	extFLAC = "flac"
	extMP4  = "mp4"
	extMPEG = "mpeg"
	extMPGA = "mpga"
	extM4A  = "m4a"
	extWEBM = "webm"
)

var downsampleOGGExt = map[string]struct{}{
	extMP3: {},
	extAAC: {},
	extWAV: {},
}

func ExtAllowedDownsampleOGG(ext string) bool {
	if _, ok := downsampleOGGExt[ext]; !ok {
		return false
	}
	return true
}

type DownsampleStrategyOption string

const (
	// The auto_best strategy will attempt to get the best quality file that it
	// can without exceeding the size cap and without upsampling. If it cannot
	// shrink to size cap even with max compression then an ErrSizeCapExceeded
	// error is returned.
	DownsampleStrategyAutoBest DownsampleStrategyOption = "auto_best"
)

const (
	bitrate48k int64 = 48000 // high quality
	bitrate24k int64 = 24000 // medium quality
	bitrate12k int64 = 12000 // low quality
)

type DownsampleOGGConfig struct {
	OutputBasePath string                   // base path for the final output target file
	SizeCap        int64                    // max allowed size of target file
	Strategy       DownsampleStrategyOption // strategy for the downsampling steps
}

func (config DownsampleOGGConfig) Validate() error {
	errs := []error{}

	basePathInfo, err := os.Stat(config.OutputBasePath)
	if err != nil {
		errs = append(errs, fmt.Errorf("invalid OutputBasePath [%v]: %v", config.OutputBasePath, err))
	}

	if !basePathInfo.IsDir() {
		errs = append(errs, fmt.Errorf("invalid OutputBasePath [%v]: not a directory", config.OutputBasePath))
	}

	if config.SizeCap == 0 {
		errs = append(errs, fmt.Errorf("invalid SizeCap [%v]: must be greater than 0", config.SizeCap))
	}

	if config.Strategy != DownsampleStrategyAutoBest {
		errs = append(errs, fmt.Errorf("invalid Strategy [%v]: not recognized", config.Strategy))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// DownsampleOGG executes a transformation of the source media file based on the config
// options, and returns a new target media wrapper for the output file. This operation
// will handle the creation and cleanup of its own unique work directory based on the
// system os.Temp. The final output file will be created in the DownsampleOGGConfig.OutputBasePath
// directory defined by the caller and it is the callers responsibility to remove it.
func (sourceMedia Media) DownsampleOGG(ctx context.Context, debug bool, config DownsampleOGGConfig) (Media, error) {

	var targetMedia Media

	if !sourceMedia.initialized {
		return targetMedia, ErrValidation{
			Err: fmt.Errorf("media uninitialized: use media constructor"),
		}
	}

	if err := config.Validate(); err != nil {
		return targetMedia, ErrValidation{
			Err: fmt.Errorf("bad config: %v", err),
		}
	}

	if !ExtAllowedDownsampleOGG(sourceMedia.GetExt()) {
		// TODO: Error handling, don't double wrap, implement err at public func
		return targetMedia, ErrValidation{
			Err: fmt.Errorf("ext not allowed: %v", sourceMedia.GetExt()),
		}
	}

	workDir, err := os.MkdirTemp("", "spiritor")
	if err != nil {
		return targetMedia, ErrFileOp{
			Err: fmt.Errorf("failed to create work dir: %v", err),
		}
	}
	defer os.RemoveAll(workDir)

	sourceFilePath := sourceMedia.GetPath()
	targetFilePath := filepath.Join(workDir, fmt.Sprintf("%v.%v", sourceMedia.GetName(), extOGG))
	finalFilePath := filepath.Join(config.OutputBasePath, fmt.Sprintf("%v.%v", sourceMedia.GetName(), extOGG))
	targetBitrate := calculateBestBitrate(sourceMedia, config.SizeCap)

	if debug {
		fmt.Printf("target bitrate for %v: %v\n", sourceMedia.GetName(), targetBitrate)
	}

	if err := ffmpeg.DownsampleOpus(sourceFilePath, targetFilePath, targetBitrate); err != nil {
		return targetMedia, ErrFileOp{
			Err: fmt.Errorf("ffmpeg failed: %v", err),
		}
	}

	targetMedia, err = NewMedia(debug, targetFilePath)
	if err != nil {
		return targetMedia, fmt.Errorf("new target media failed: %w", err)
	}

	if targetMedia.GetSize() > config.SizeCap {
		return targetMedia, ErrSizeCapExceeded{
			SizeCap:  config.SizeCap,
			FileSize: targetMedia.GetSize(),
		}
	}

	if err := os.Rename(targetFilePath, finalFilePath); err != nil {
		return targetMedia, ErrFileOp{
			Err: fmt.Errorf("file move/rename failed: %v", err),
		}
	}

	targetMedia, err = NewMedia(debug, finalFilePath)
	if err != nil {
		return targetMedia, fmt.Errorf("new final media failed: %w", err)
	}

	return targetMedia, nil
}

// calculateBestBitrate will attempt to calculate the highest possible bitrate we
// can downsample the media to while staying under the size cap. This algorithm is
// based on the information from ffmpeg probe and calculations can be thrown off in
// some situations such as when the encoding uses variable bitrate which can throw
// off the file duration which we use for file size projections. If the filesize for
// the lowest available bitrate still exceeds the size cap then this func will just
// return that lowest bitrate so the caller can attempt it. The caller should perform
// a final check of the actual size of the file after it is downsampled. This func
// will return a string such as "24k" which can be used directly with ffmpeg.
func calculateBestBitrate(sourceMedia Media, sizeCap int64) string {
	/*
		example_signal.aac (07:12m, 432s)
		size:1738968
		duration:376186258000
		bitrate:36981

		example_signal.aac.ogg
		size:1304626
		duration:404196271000
		bitrate:0

		-----

		example_zoom.MP3 (45:05m, 2705s)
		size:37865691
		duration:2704692214000
		bitrate:112000

		example_zoom.MP3.ogg
		size:7992132
		duration:2704698750000
		bitrate:0

		-----

		example_zoom.wav (4:07m, 247s)
		size:7891984
		duration:246622062000
		bitrate:256000

		example_zoom.wav.ogg
		size:708891
		duration:246628562000
		bitrate:0
	*/

	const (
		str48k = "48k"
		str24k = "24k"
		str12k = "12k"
	)

	// To determine the file size of an audio file, we have to multiply the
	// bit rate of the audio by its duration in seconds. Divide that final
	// number by 8 to get the size in bytes rather than bits.

	bitrate := int64(sourceMedia.bitrate)
	seconds := int64(math.Round(sourceMedia.duration.Seconds()))

	// if either bitrate or seconds are zero then we cannot perform projections
	// so just return the lowest bitrate
	if bitrate == 0 || seconds == 0 {
		return str12k
	}

	// Important: The checking order from highest to lowest is important
	// in that it prevents accidental upsampling.

	if bitrate > bitrate48k {
		if bitrate48k*seconds/8 < sizeCap {
			return str48k
		}
	}

	if bitrate > bitrate24k {
		if bitrate24k*seconds/8 < sizeCap {
			return str24k
		}
	}

	// we don't need to do a projection for 12k since returning this is our
	// default condition regardless of whether it falls within the size cap
	return str12k
}
