package stream

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strconv"
)

// Profile descreve um perfil de transcodificação.
type Profile struct {
	Codec     string // libmp3lame, libopus, aac
	Container string // mp3, ogg, adts
	MIME      string
}

var profiles = map[string]Profile{
	"mp3":  {Codec: "libmp3lame", Container: "mp3", MIME: "audio/mpeg"},
	"opus": {Codec: "libopus", Container: "ogg", MIME: "audio/ogg"},
	"aac":  {Codec: "aac", Container: "adts", MIME: "audio/aac"},
}

// SupportedFormat informa se um formato de transcodificação é conhecido.
func SupportedFormat(format string) bool {
	_, ok := profiles[format]
	return ok
}

// Transcoder gerencia transcodificação sob demanda via ffmpeg.
type Transcoder struct {
	ffmpeg string
}

func NewTranscoder(ffmpegPath string) *Transcoder { return &Transcoder{ffmpeg: ffmpegPath} }

// Stream transcodifica `path` para `format`/`maxBitRate` e escreve em w como
// resposta chunked (não-seekável). Para seek, o cliente reabre com timeOffset.
// O processo ffmpeg é encerrado quando o contexto da requisição é cancelado
// (cliente desconectou), evitando processos zumbis.
func (t *Transcoder) Stream(ctx context.Context, w http.ResponseWriter, path, format string, maxBitRate, timeOffset int) error {
	prof, ok := profiles[format]
	if !ok {
		return fmt.Errorf("formato não suportado: %s", format)
	}
	bitrate := maxBitRate
	if bitrate <= 0 {
		bitrate = 192
	}

	args := []string{"-hide_banner", "-loglevel", "error"}
	if timeOffset > 0 {
		args = append(args, "-ss", strconv.Itoa(timeOffset))
	}
	args = append(args,
		"-i", path,
		"-map", "0:a:0",
		"-c:a", prof.Codec,
		"-b:a", strconv.Itoa(bitrate)+"k",
		"-vn",
		"-f", prof.Container,
		"pipe:1",
	)

	cmd := exec.CommandContext(ctx, t.ffmpeg, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	w.Header().Set("Content-Type", prof.MIME)
	w.Header().Set("Accept-Ranges", "none")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)

	flusher, _ := w.(http.Flusher)
	buf := make([]byte, 32*1024)
	for {
		n, rerr := stdout.Read(buf)
		if n > 0 {
			if _, werr := w.Write(buf[:n]); werr != nil {
				break // cliente desconectou
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
		if rerr != nil {
			if rerr != io.EOF {
				err = rerr
			}
			break
		}
	}
	// Garante término do processo (CommandContext mata ao cancelar; Wait coleta).
	_ = cmd.Wait()
	return err
}
