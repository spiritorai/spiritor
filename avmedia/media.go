package avmedia

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spiritorai/spiritor/ffmpeg"
)

// NewMedia should always be used to initialize a new media struct from outside
// of the avtools package
func NewMedia(debug bool, filePath string) (Media, error) {

	if debug {
		fmt.Printf("new media: %v\n", filePath)

		dump, err := ffmpeg.ProbeDump(filePath)
		if err != nil {
			fmt.Printf("ffprobe dump failed: %v\n", err)
		}
		fmt.Printf("ffprobe dump: %v\n", dump)
	}

	var media Media

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return media, ErrFileOp{
			Err: fmt.Errorf("failed to stat file: %v", err),
		}
	}

	media.path = filePath
	media.size = fileInfo.Size()

	bitrate, err := ffmpeg.ProbeBitrate(filePath)
	if err != nil {
		return media, ErrFileOp{
			Err: fmt.Errorf("failed to probe bitrate: %v", err),
		}
	}

	media.bitrate = bitrate

	duration, err := ffmpeg.ProbeDuration(filePath)
	if err != nil {
		return media, ErrFileOp{
			Err: fmt.Errorf("failed to probe duration: %v", err),
		}
	}

	media.duration = duration

	media.initialized = true

	if debug {
		fmt.Printf("media file created: %+v\n", media)
	}

	return media, nil
}

// Media is a wrapper for an AV file which enables fast lookups of essential
// properties as well as common transformations.
// IMPORTANT: These structs are wrappers for file paths and metadata only, and
// should never be updated to contain references to file contents or stream pointers.
// It should always be safe for the caller to drop the struct references and allow them
// to be garbage collected without having to perform any cleanup tasks, for example
// closing a reader.
type Media struct {
	path        string        // absolute file path, eg: /my/docs/zoom.mp3
	size        int64         // file size in bytes
	duration    time.Duration // playback length over time
	bitrate     int           // encoded bitrate
	initialized bool          // internal tracking for properly initialized media
}

// absolute file path, eg: /my/docs/zoom.mp3
func (f Media) GetPath() string {
	return f.path
}

// base directory, eg: /my/docs/
func (f Media) GetDir() string {
	return filepath.Dir(f.path)
}

// file name, eg: zoom.mp3
func (f Media) GetName() string {
	return filepath.Base(f.path)
}

// file extension, eg: mp3
func (f Media) GetExt() string {
	return strings.ToLower(strings.TrimLeft(filepath.Ext(f.path), "."))
}

// file size in bytes
func (f Media) GetSize() int64 {
	return f.size
}

// playback length over time
func (f Media) GetDuration() time.Duration {
	return f.duration
}

// encoded bitrate
func (f Media) GetBitrate() int {
	return f.bitrate
}
