package macsayer

import (
	"crypto/md5"
	"fmt"
	"musayer/macsayer/musayerapi"
	"os"
	"os/exec"
	"path"
	"strings"
)

// MacSayer is a wrapper of macSay to implement musayerapi.Sayer.
type MacSayer struct {
	// the format (extension name) of the output audio file. Default is "aiff".
	Format string
	// CleanMode indicates whether to:
	//  - clean the temporary file when the Say method is returned.
	//  - clean the temporary directory when the MacSayer is closed.
	// Default is false.
	CleanMode bool

	// the temporary directory to store the output audio file. Will be created automatically. DO NOT CHANGE THIS VALUE.
	tmpdir string

	// MacSayer implements musayerapi.Sayer
	musayerapi.Sayer
}

// NewMacSayer creates a new MacSayer.
//
// Available options are WithVoice, WithRate and WithAudioDevice.
func NewMacSayer(opts ...MacSayerOption) (*MacSayer, error) {
	tmpdir, err := os.MkdirTemp("", "macsayer")
	if err != nil {
		return nil, err
	}
	s := &MacSayer{
		Format: "aiff",
		tmpdir: tmpdir,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s, nil
}

// MacSayerOption configures a MacSayer
//
// Available options are WithVoice, WithRate and WithAudioDevice.
type MacSayerOption func(s *MacSayer)

func WithFormat(format string) MacSayerOption {
	return func(s *MacSayer) {
		s.Format = format
	}
}

func WithClean(c bool) MacSayerOption {
	return func(s *MacSayer) {
		s.CleanMode = c
	}
}

// Say calls SAY(1) to convert text to audible speech via apple's Speech Synthesis manager.
//
// the role should be in the format of one of the following:
//
//	"{voice}"          // say -v {voice}
//	"{voice}:{rate}"   // say -v {voice} -r {rate}
func (s MacSayer) Say(role string, text string) (format string, audio []byte, err error) {
	// say -v {s.Voice} -r {s.Rate} -o {outputdir}/{hash}.{format} {s.Text}

	role = strings.TrimSpace(role)
	text = strings.TrimSpace(text)
	if text == "" {
		return "", nil, nil
	}

	mc := macSay{
		text:      text,
		outputdir: s.tmpdir,
		format:    s.Format,
	}

	// parse role
	rs := strings.Split(role, ":")
	if len(rs) >= 1 {
		mc.voice = rs[0]
	}
	if len(rs) >= 2 {
		mc.rate = rs[1]
	}

	// say
	err = mc.say()
	if err != nil {
		return "", nil, err
	}
	defer func() {
		if s.CleanMode {
			os.Remove(mc.outputFile())
		}
	}()

	// read
	audio, err = os.ReadFile(mc.outputFile())
	if err != nil {
		return "", nil, err
	}

	return mc.format, audio, nil
}

// Close closes the MacSayer
func (s MacSayer) Close() error {
	if !s.CleanMode {
		return nil
	}
	return os.RemoveAll(s.tmpdir)
}

// macSay is a wrapper of a SAY(1) command call.
type macSay struct {
	voice     string
	rate      string
	format    string
	outputdir string
	text      string
}

// outputFile returns the output file path:
//
//	{outputdir}/{hash}.{format}
func (s macSay) outputFile() string {
	return path.Join(s.outputdir, s.hash()+"."+s.format)
}

func (s macSay) hash() string {
	h := md5.New()
	h.Write([]byte(s.voice))
	h.Write([]byte(s.rate))
	h.Write([]byte(s.format))
	h.Write([]byte(s.text))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// args returns the arguments for SAY(1)
func (s macSay) args() []string {
	args := make([]string, 0, 3)

	// must: output
	args = append(args, "-o", s.outputFile())

	// options
	if s.voice != "" {
		args = append(args, "-v", s.voice)
	}
	if s.rate != "" {
		args = append(args, "-r", s.rate)
	}

	// must: text
	args = append(args, s.text)

	return args
}

// say calls SAY(1) with s.args()
func (s macSay) say() error {
	args := s.args()
	cmd := exec.Command("say", args...)
	return cmd.Run()
}
