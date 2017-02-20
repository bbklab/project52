package ffmpeg

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	ffprobe "github.com/vansante/go-ffprobe"
	"github.com/xfrr/goffmpeg/ffmpeg"
	"github.com/xfrr/goffmpeg/models"
	"github.com/xfrr/goffmpeg/transcoder"
)

type Handler interface {
	Probe(file string) (*ProbeData, error)
	ExtractJpg(file string) (string, error)
	ExtractGif(file string) (string, error)
	ToHLS(in, out string, encrypt bool) (<-chan models.Progress, <-chan error, error)
}

type handler struct{}

func NewHandler() Handler {
	return &handler{}
}

type ProbeData struct {
	Size            float64       `json:"size"`
	Duration        time.Duration `json:"duration"`
	VideoResolution string        `json:"video_resolution"`
	VideoCodec      string        `json:"video_codec"`
	AudioCodec      string        `json:"audio_codec"`
}

func (h *handler) Probe(file string) (*ProbeData, error) {
	probin, err := exec.LookPath("ffprobe")
	if err != nil {
		return nil, err
	}
	ffprobe.SetFFProbeBinPath(probin)

	info, err := ffprobe.GetProbeData(file, time.Second*10)
	if err != nil {
		return nil, err
	}

	var (
		audio   = info.GetFirstAudioStream()
		video   = info.GetFirstVideoStream()
		format  = info.Format
		size, _ = strconv.ParseFloat(format.Size, 10)
	)

	data := &ProbeData{
		Size:     size,
		Duration: format.Duration(),
	}
	if video != nil {
		data.VideoResolution = fmt.Sprintf("%dx%d", video.Width, video.Height)
		data.VideoCodec = video.CodecName
	}
	if audio != nil {
		data.AudioCodec = audio.CodecName
	}

	return data, nil
}

func (h *handler) ExtractJpg(file string) (string, error) {
	ffbin, err := exec.LookPath("ffmpeg")
	if err != nil {
		return "", err
	}

	// get the video time duration by seconds
	info, err := h.Probe(file)
	if err != nil {
		return "", err
	}

	secs := int(info.Duration.Seconds())
	mid := fmt.Sprintf("%d", secs/2)

	jpg := strings.TrimSuffix(file, filepath.Ext(file)) + ".jpg"
	args := []string{"-y", "-ss", mid, "-i", file, "-frames", "1", jpg}
	_, stderr, err := runCmd(nil, ffbin, args...)
	if err != nil {
		return "", fmt.Errorf("extract media middle jpg error: %v - [%s]", err, stderr)
	}

	return jpg, nil
}

func (h *handler) ExtractGif(file string) (string, error) {
	ffbin, err := exec.LookPath("ffmpeg")
	if err != nil {
		return "", err
	}

	// get the video time duration by seconds
	info, err := h.Probe(file)
	if err != nil {
		return "", err
	}

	// get the head/middle/tail sections
	parts := make([][2]string, 0, 0)
	secs := int(info.Duration.Seconds())
	if secs < 30 {
		parts = append(parts, [2]string{"1", strconv.Itoa(secs)})
	} else {
		divsize := secs / 3
		if divsize-10 > 10 {
			parts = append(parts, [2]string{"1", "10"})
		}
		parts = append(parts, [2]string{strconv.Itoa(divsize - 10), strconv.Itoa(divsize)})
		parts = append(parts, [2]string{strconv.Itoa(divsize*2 - 10), strconv.Itoa(divsize * 2)})
		parts = append(parts, [2]string{strconv.Itoa(divsize*3 - 10), strconv.Itoa(divsize * 3)})
	}

	// make temp directory to hold the jpg images
	jpgdir, err := ioutil.TempDir(os.TempDir(), "ffdemo.ffextract.jpg.")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(jpgdir)

	// extract each sections to jpg images
	for idx, part := range parts {
		from, to := part[0], part[1]
		args := []string{"-y", "-ss", from, "-to", to, "-i", file, "-r", "0.5", "-f", "image2", "-s", "320x240", jpgdir + "/" + fmt.Sprintf("0-%d-", idx) + `%3d.jpg`}
		_, stderr, err := runCmd(nil, ffbin, args...)
		if err != nil {
			return "", fmt.Errorf("extract section %d media jpg error: %v - [%s]", idx, err, stderr)
		}
	}

	// combine jpg images to gif
	gif := strings.TrimSuffix(file, filepath.Ext(file)) + ".gif"
	args := []string{"-f", "image2", "-pattern_type", "glob", "-framerate", "5", "-y", "-i", jpgdir + `/*.jpg`, gif}
	_, stderr, err := runCmd(nil, ffbin, args...)
	if err != nil {
		return "", fmt.Errorf("combine media jpg as gif error: %v - [%s]", err, stderr)
	}

	return gif, nil
}

func (h *handler) ToHLS(in, out string, encrypt bool) (<-chan models.Progress, <-chan error, error) {
	ffbin, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, nil, err
	}

	probin, err := exec.LookPath("ffprobe")
	if err != nil {
		return nil, nil, err
	}

	trans := new(transcoder.Transcoder)
	trans.SetConfiguration(ffmpeg.Configuration{
		FfmpegBin:  ffbin,
		FfprobeBin: probin,
	})

	err = trans.Initialize(in, out)
	if err != nil {
		return nil, nil, err
	}

	trans.MediaFile().SetVideoCodec("libx264")                               // -c:v
	trans.MediaFile().SetAudioCodec("copy")                                  // -c:a
	trans.MediaFile().SetOutputFormat("hls")                                 // -f
	trans.MediaFile().SetHlsSegmentDuration(2)                               // -hls_time
	trans.MediaFile().SetHlsListSize(0)                                      // -hls_list_size
	trans.MediaFile().SetHlsPlaylistType("vod")                              // -hls_playlist_type
	filets := path.Dir(out) + "/" + strings.ToUpper(randgen(6)) + `-%04d.ts` // xxxx-%04d.ts
	trans.MediaFile().SetHlsSegmentFilename(filets)                          // -hls_segment_filename

	if encrypt {
		fkey := path.Join(path.Dir(out), randgen(8)+".key")         // `xxxxxxx.key` under same dir of output file
		fkeyinfo := path.Join(path.Dir(out), randgen(8)+".keyinfo") // `xxxxxxx.keyinfo` under same dir of output file
		if err = genKeyFile(fkey, fkeyinfo); err != nil {
			return nil, nil, err
		}
		trans.MediaFile().SetHlsKeyInfoFile(fkeyinfo) // -hls_key_info_file
	}

	donech := trans.Run(true) // with progress
	progch := trans.Output()  // progress channel

	return progch, donech, nil
}

// just similar as following commands
/*
  openssl rand  16 > video.key
  echo video.key                > video.keyinfo
  echo video.key                >> video.keyinfo
  echo $(openssl rand -hex 16)  >> video.keyinfo
*/
func genKeyFile(fkey, fkeyinfo string) error {
	key := randgen(16)
	err := ioutil.WriteFile(fkey, []byte(key), os.FileMode(0644))
	if err != nil {
		return err
	}
	iv := randgen(32)
	keyinfo := fmt.Sprintf("%s\r\n%s\r\n%s\r\n", path.Base(fkey), fkey, iv)
	return ioutil.WriteFile(fkeyinfo, []byte(keyinfo), os.FileMode(0644))
}

func randgen(size int) string {
	id := make([]byte, size)
	io.ReadFull(rand.Reader, id)
	return hex.EncodeToString(id)[:size]
}

func runCmd(envs map[string]string, cmd string, args ...string) (string, string, error) {
	var (
		outbuf bytes.Buffer
		errbuf bytes.Buffer
	)

	command := exec.Command(cmd, args...)
	for key, val := range envs {
		command.Env = append(command.Env, fmt.Sprintf("%s=%s", key, val))
	}
	command.Stdout = &outbuf
	command.Stderr = &errbuf

	err := command.Run()
	return outbuf.String(), errbuf.String(), err

}
