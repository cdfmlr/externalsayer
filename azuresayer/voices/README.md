# azure voices

- [`voices-list.json`](./voices-list.json): list of voices, from `curl https://eastus.tts.speech.microsoft.com/cognitiveservices/voices/list --header 'Ocp-Apim-Subscription-Key: xxx'`
- [`jenny.xml`](./jenny.xml): a voiceTemplate.
- [`xmlToSingleLine.py`](./xmlToSingleLine.py): convert xml to single line.

You drop your voice ssml here (the text should be left as `{{.}}`), and use `xmlToSingleLine.py` to convert it to single line and then use it in your config.
