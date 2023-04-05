package azuresayer

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"net/http"
	"sync"
	"text/template"
)

// AzureSayer convert text to speech via Microsoft Azure Speech API.
type AzureSayer struct {
	SpeechKey    string
	SpeechRegion string
	Roles        map[string]string // role -> voiceTemplate (that is ssml with %s for the text to be spoken)

	FormatMicrosoft   string // "audio-16khz-32kbitrate-mono-mp3"
	FormatMimeSubtype string // mp3

	Mutex sync.RWMutex
}

// NewAzureSayer returns a new AzureSayer with the given speechKey and speechRegion.
// Format will be set to "mp3" by default. And the Roles will be empty.
//
// Roles must be set before calling Say. Example:
//
//	sayer := azuresayer.NewAzureSayer("key", "region")
//	sayer.mutex.Lock()
//	sayer.Roles["Jenny"] = `<speak version='1.0' xml:lang='en-US'><voice name='en-US-JennyNeural'>{{.}}</voice></speak>`
//	sayer.mutex.Unlock()
//	format, audio, err := sayer.Say("Jenny", "hello world")
func NewAzureSayer(speechKey string, speechRegion string) *AzureSayer {
	return &AzureSayer{
		SpeechKey:         speechKey,
		SpeechRegion:      speechRegion,
		Roles:             map[string]string{},
		FormatMicrosoft:   "audio-16khz-32kbitrate-mono-mp3",
		FormatMimeSubtype: "mp3",
	}
}

// Say implements the Sayer interface. Say accquires the read lock,
// maps the role to a voiceTemplate ssml, and calls say to do TTS.
func (s *AzureSayer) Say(role string, text string) (format string, audio []byte, err error) {
	s.Mutex.RLock()
	defer s.Mutex.RUnlock()

	voiceTemplate, ok := s.Roles[role]
	if !ok {
		return "", nil, fmt.Errorf("unknown role: %s", role)
	}

	audio, err = s.say(voiceTemplate, text, s.FormatMicrosoft)
	return s.FormatMimeSubtype, audio, err
}

// say uses the Azure Speech API to convert text to speech.
func (s *AzureSayer) say(voiceTemplate string, text string, format string) (audio []byte, err error) {
	// 	curl --location --request POST "https://${SPEECH_REGION}.tts.speech.microsoft.com/cognitiveservices/v1" ^
	// 	--header "Ocp-Apim-Subscription-Key: ${SPEECH_KEY}" ^
	// 	--header 'Content-Type: application/ssml+xml' ^
	// 	--header 'X-Microsoft-OutputFormat: audio-16khz-128kbitrate-mono-mp3' ^
	// 	--header 'User-Agent: curl' ^
	// 	--data-raw '<speak version='\''1.0'\'' xml:lang='\''en-US'\''>
	// 	    <voice xml:lang='\''en-US'\'' xml:gender='\''Female'\'' name='\''en-US-JennyNeural'\''>
	// 	        my voice is my passport verify me
	// 	    </voice>
	// 	</speak>' > output.mp3

	url := fmt.Sprintf("https://%s.tts.speech.microsoft.com/cognitiveservices/v1",
		s.SpeechRegion)

	ssml, err := template.New("ssml").Parse(voiceTemplate)
	if err != nil {
		return nil, err
	}

	var body bytes.Buffer
	err = ssml.Execute(&body, html.EscapeString(text))
	if err != nil {
		return nil, err
	}
	fmt.Printf("ssml: \n\t%s\n", body.String())

	req, err := http.NewRequest("POST", url, bytes.NewReader(body.Bytes()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Ocp-Apim-Subscription-Key", s.SpeechKey)
	req.Header.Set("Content-Type", "application/ssml+xml")
	req.Header.Set("X-Microsoft-Outputformat", "audio-16khz-32kbitrate-mono-mp3")
	req.Header.Set("User-Agent", "muli/externalsayer")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	audio, err = io.ReadAll(resp.Body)
	return audio, err
}
