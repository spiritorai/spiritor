# Spiritor AI

[![Go Report Card](https://goreportcard.com/badge/github.com/spiritorai/spiritor)](https://goreportcard.com/report/github.com/spiritorai/spiritor)
[![Go Reference](https://pkg.go.dev/badge/github.com/spiritorai/spiritor.svg)](https://pkg.go.dev/github.com/spiritorai/spiritor)

Spiritor is a privacy centric and open source toolkit for writers and content creators utilizing speech-to-text transcription and ai. Think of it as your creative copilot. Rapidly generate content in your own voice and based on your own ideas for:

* Essays, books and articles
* Video scripts
* Podcasts
* Marketing copy

## Current Features

* Batch speech-to-text transcription of audio files:
	* Currently supported: `mp3`, `aac`, `wav`
	* Utilizing [OpenAI Whisper API](https://platform.openai.com/docs/guides/speech-to-text)
	* Large batch processing
	* Large file (<= 3 hours) support
	* Optimized downsampling (no file splitting)

## Roadmap Features

* Multiple speakers identification
* Super large file (> 3 hours) support (via file splitting)
* Support additional audio formats
* Speech-to-text transcription of video files
* Transcript-based editing of audio/video files
* AI assisted recomposition of transcripts and writing
* Text-to-speech content generation

## Data Retention Policy

Spiritor itself does not retain any data. The transcription engine utilizes a pass-through to the [OpenAI Whisper API](https://platform.openai.com/docs/guides/speech-to-text). You must configure the CLI with your own OpenAI API key.

Here is a summary of the [data retention policy of OpenAI](https://platform.openai.com/docs/models/how-we-use-your-data) which applies to consumer use of their Whisper API:

> Your data is your data.
>
> As of March 1, 2023, data sent to the OpenAI API will not be used to train or improve OpenAI models (unless you explicitly opt in). One advantage to opting in is that the models may get better at your use case over time.
> 
> To help identify abuse, API data may be retained for up to 30 days, after which it will be deleted (unless otherwise required by law). For trusted customers with sensitive applications, zero data retention may be available. With zero data retention, request and response bodies are not persisted to any logging mechanism and exist only in memory in order to serve the request.

# CLI Usage

All features of spiritor are accessible via a CLI (command line interface) that can be installed and run from your desktop computer or laptop. The following operating systems are supported:

* Linux
* MacOS

Windows support is on the roadmap.

## Dependencies

While the CLI aims to be self-contained into a single binary, we are not quite there yet. At this time, the following external dependencies must be installed on your machine separately:

* [ffmpeg](https://ffmpeg.org/download.html)

## Installation

Simply download the latest pre-compiled binary/executable for your operating system. Then add the binary to your system path so that it can be executed from any folder.

** NOTE: The binaries are not currently available for download but are being implemented. In the meantime you can just clone the package and `go install spiritor.go` to install locally. **

## Commands

Once installed you can access the CLI through the root command `spiritor`, eg:

```sh
spiritor --help
```

For any commands which rely on OpenAI you will need to export the `API_KEY_OPENAI` env var. This can be done on each separate command like so: 

```sh
API_KEY_OPENAI=abc123 && spiritor foo
API_KEY_OPENAI=abc123 && spiritor bar
```

Or exported once and available to all subsequent commands like so:

```sh
export API_KEY_OPENAI=abc123
spiritor foo
spiritor bar
```

### Scribe

The `scribe` command will transcribe the indicated file(s) and write the results back to the directory where the files live, utilizing the same file name with an additional extension of the format appended. For example:

```sh
spiritor scribe path/to/my.mp3
>> outputs: path/to/my.mp3.txt
```

Common path tokens such as `*` are recognized, and the specified path is always relative to the directory where you execute the command. The most common use case would to `cd` into the directory which contains a batch of audio audio files and:

```sh
# Target all mp3 files in the folder
spiritor scribe *.mp3

# Target all supported file types in the folder (unsupported types and files with existing transcripts will be skipped)
spiritor scribe *.*
```

It is safe to re-run these batch commands on a folder where the contents have changed. If a transcript already exists for any files then it will be skipped, unless you specify the `-f` param which will force it to be regenerated, eg:

```sh
# Target all supported file types in the folder (existing transcripts will be overwritten)
spiritor scribe *.* -f
```

Full command details can be obtained via `spiritor scribe --help`.

#### Large Batches

When executing a large batch of files with a single command, spiritor will process multiple files in parallel by spawning multiple workers both for the downsampling and the transcription processes. These worker pools are currently not optimized for all systems and are prone to bugs (in particular failures when the whisper api gets too many concurrent requests), which means that when processing large batches you may get errors for some (but not all) of the files. If this happens then simply run the command again (without the `-f` flag) and it should successfully process any of the files that failed in the first run.

#### Large Audio Files

The Whisper API has a 25mb limit on audio file size, and larger audio files will be rejected. The common strategy for dealing with this is to split large files into smaller chunks, transcribe each chunk separately, and then re-combine the transcripts back into a single file. However the common problem with this strategy is that file splitting can cause problems with the transcription grammar and sentence structure.

Spiritor uses a different approach to deal with the 25mb limit. Rather than splitting the audio file we downsample it into a smaller file before sending them up for transcription, and thereby circumvent the file splitting problems. The downsampling algorithm we use is able to shrink each file depending on it's format significantly and without causing any noticeable impact to the transcription quality. This means that we can shrink a 50-minute mp3 file of 38mb down to an 8mb file of equivalent quality for transcription.

The downside to this downsample method is that we do reach a bottom limit where if we are unable to shrink the file down below 25mb while still retaining optimal quality then we cannot transcribe that file. However, you should not ever hit this limit unless your audio is 3+ hours in duration. Super long files like this will be supported in the future through additional strategies such as file splitting combined with downsampling.

### Debug Mode

All commands will support a `--debug` flag which will enable detailed console output. You may be required to copy and paste the full debug output when submitting a new issue.

# Developer Notes

Golang developers may import and utilize the spiritor module directly. If you do so please be sure to use vendoring, because at this point in development the package organization and function signatures are liable to change.

Contributors please see: [DEVELOPMENT.md](DEVELOPMENT.md).
