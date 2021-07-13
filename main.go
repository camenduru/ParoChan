package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"text/template"
	"time"

	"github.com/Stanly1995/gtranslate"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/comprehend"
	"github.com/aws/aws-sdk-go/service/polly"
	"github.com/bwmarrin/discordgo"
	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/fsnotify/fsnotify"
	"github.com/gempir/go-twitch-irc"
	"github.com/getlantern/systray"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"mvdan.cc/xurls/v2"

	icon "./icons"
	detector1 "github.com/abadojack/whatlanggo"
	detector2 "github.com/wmentor/lang"
)

var (
	dashboardTemplate = template.Must(template.ParseFiles("speech.html"))
	settings          map[string]string
	readthis          = ""
	lang              = "en"
	input             *polly.SynthesizeSpeechInput
	ignoredusers      []string
	ignoredwords      []string
	regexs            map[string]string
	voices            map[string]string
	rootkeys          map[string]string
)

func onReady() {

	getFiles()

	go watch()

	systray.SetIcon(icon.Data)
	systray.SetTitle("Twitch Parrot (ParoChan) v1.6 (dev:camenduru)")
	systray.SetTooltip("Twitch Parrot (ParoChan) v1.6 (dev:camenduru)")

	exe, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exePath := filepath.Dir(exe)
	mSettings := systray.AddMenuItem("Settings", "settings.txt")
	systray.AddSeparator()
	mIgnoredUsers := systray.AddMenuItem("Ignored Users", "ignoredusers.txt")
	systray.AddSeparator()
	mIgnoredWords := systray.AddMenuItem("Ignored Words", "ignoredwords.txt")
	systray.AddSeparator()
	mRegexList := systray.AddMenuItem("Regex List", "regexlist.txt")
	systray.AddSeparator()
	mVoiceList := systray.AddMenuItem("Voice List", "voicelist.txt")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit")

	go func() {
		for {
			select {
			case <-mSettings.ClickedCh:
				exec.Command(`notepad`, exePath+`\settings.txt`).Run()
			case <-mIgnoredUsers.ClickedCh:
				exec.Command(`notepad`, exePath+`\ignoredusers.txt`).Run()
			case <-mIgnoredWords.ClickedCh:
				exec.Command(`notepad`, exePath+`\ignoredwords.txt`).Run()
			case <-mRegexList.ClickedCh:
				exec.Command(`notepad`, exePath+`\regexlist.txt`).Run()
			case <-mVoiceList.ClickedCh:
				exec.Command(`notepad`, exePath+`\voicelist.txt`).Run()
			case <-mQuit.ClickedCh:
				systray.Quit()
			}
		}
	}()

	if strings.Compare(settings["speechlanguage"], "off") != 0 {
		go speech()
	}

	if strings.Compare(settings["discord-token"], "") != 0 {
		go discord(settings["discord-token"])
	}

	if strings.Compare(settings["oauth"], "") != 0 {
		go task()
	}
}

func watch() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()
	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				log.Println("event:", event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("modified file:", event.Name)
					systray.Quit()
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()
	err = watcher.Add("settings.txt")
	err = watcher.Add("ignoredusers.txt")
	err = watcher.Add("ignoredwords.txt")
	err = watcher.Add("regexlist.txt")
	err = watcher.Add("voicelist.txt")
	if err != nil {
		log.Fatal(err)
	}
	<-done
}

func getFiles() {
	// settings.txt
	settings = make(map[string]string)
	settingsTxt, errSettings := os.Open("settings.txt")
	if errSettings != nil {
		log.Fatal(errSettings)
	}
	defer settingsTxt.Close()
	settingsScanner := bufio.NewScanner(settingsTxt)
	for settingsScanner.Scan() {
		settingKeyValue := strings.Split(string(settingsScanner.Text()), "~")
		settings[strings.TrimSpace(settingKeyValue[0])] = strings.TrimSpace(settingKeyValue[1])
	}
	if errSettingsScanner := settingsScanner.Err(); errSettingsScanner != nil {
		log.Fatal(errSettingsScanner)
	}

	// regexlist.txt
	regexs = make(map[string]string)
	regexListTxt, errRegexList := os.Open("regexlist.txt")
	if errRegexList != nil {
		log.Fatal(errRegexList)
	}
	defer regexListTxt.Close()
	regexScanner := bufio.NewScanner(regexListTxt)
	for regexScanner.Scan() {
		regexKeyValue := strings.Split(string(regexScanner.Text()), "~")
		regexs[strings.TrimSpace(regexKeyValue[0])] = strings.TrimSpace(regexKeyValue[1])
	}
	if errRegexScanner := regexScanner.Err(); errRegexScanner != nil {
		log.Fatal(errRegexScanner)
	}

	// rootkey.csv
	rootkeys = make(map[string]string)
	rootkeyTxt, errRootkeys := os.Open("rootkey.csv")
	if errRootkeys != nil {
		log.Fatal(errRootkeys)
	}
	defer rootkeyTxt.Close()
	rootkeyScanner := bufio.NewScanner(rootkeyTxt)
	for rootkeyScanner.Scan() {
		rootkeyKeyValue := strings.Split(string(rootkeyScanner.Text()), "=")
		rootkeys[strings.TrimSpace(rootkeyKeyValue[0])] = strings.TrimSpace(rootkeyKeyValue[1])
	}
	if errRootkeyScanner := rootkeyScanner.Err(); errRootkeyScanner != nil {
		log.Fatal(errRootkeyScanner)
	}

	// voicelist.txt
	voices = make(map[string]string)
	voiceListTxt, errVoiceList := os.Open("voicelist.txt")
	if errVoiceList != nil {
		log.Fatal(errVoiceList)
	}
	defer voiceListTxt.Close()
	voiceScanner := bufio.NewScanner(voiceListTxt)
	for voiceScanner.Scan() {
		voiceKeyValue := strings.Split(string(voiceScanner.Text()), "=")
		voices[strings.TrimSpace(voiceKeyValue[0])] = strings.TrimSpace(voiceKeyValue[1])
	}
	if errVoiceScanner := voiceScanner.Err(); errVoiceScanner != nil {
		log.Fatal(errVoiceScanner)
	}

	// ignoredusers.txt
	ignoredUsersTxt, errIgnoredUsersTxt := os.Open("ignoredusers.txt")
	if errIgnoredUsersTxt != nil {
		log.Fatal(errIgnoredUsersTxt)
	}
	defer ignoredUsersTxt.Close()
	ignoredUsersScanner := bufio.NewScanner(ignoredUsersTxt)
	for ignoredUsersScanner.Scan() {
		ignoredusers = append(ignoredusers, strings.TrimSpace(ignoredUsersScanner.Text()))
	}
	if errIgnoredUsersScanner := ignoredUsersScanner.Err(); errIgnoredUsersScanner != nil {
		log.Fatal(errIgnoredUsersScanner)
	}

	// ignoredwords.txt
	ignoredWordsTxt, errIgnoredWordsTxt := os.Open("ignoredwords.txt")
	if errIgnoredWordsTxt != nil {
		log.Fatal(errIgnoredWordsTxt)
	}
	defer ignoredWordsTxt.Close()
	ignoredWordsScanner := bufio.NewScanner(ignoredWordsTxt)
	for ignoredWordsScanner.Scan() {
		ignoredwords = append(ignoredwords, strings.TrimSpace(ignoredWordsScanner.Text()))
	}
	if errIgnoredWordsScanner := ignoredWordsScanner.Err(); errIgnoredWordsScanner != nil {
		log.Fatal(errIgnoredWordsScanner)
	}
}

func onExit() {
}

func TruncateString(str string, length int) string {
	if length <= 0 {
		return ""
	}
	truncated := ""
	count := 0
	for _, char := range str {
		truncated += string(char)
		count++
		if count >= length {
			break
		}
	}
	return truncated
}

func GoogleSpeak(text, language string) error {
	url := fmt.Sprintf("http://translate.google.com/translate_tts?ie=UTF-8&total=1&idx=0&textlen=32&client=tw-ob&q=%s&tl=%s", url.QueryEscape(text), language)
	response, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer response.Body.Close()

	streamer, format, err := mp3.Decode(response.Body)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer streamer.Close()

	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))

	done := make(chan bool)
	speaker.Play(beep.Seq(streamer, beep.Callback(func() {
		done <- true
	})))

	<-done

	return nil
}

func main() {
	systray.Run(onReady, onExit)
}

func speech() {
	r := mux.NewRouter()
	r.Handle("/", dashboardHandler).Methods("GET")
	r.Handle("/", analysisHandler).Methods("POST")
	go open("http://localhost:8888/")
	http.ListenAndServe(":8888", handlers.LoggingHandler(os.Stdout, r))
}

var dashboardHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	err := dashboardTemplate.Execute(w, map[string]interface{}{
		"SpeechAPI":      "http://localhost:8888/",
		"SpeechLanguage": settings["speechlanguage"], //ja-JP en-US
	})
	if err != nil {
		http.Error(w, "Error loading page.", http.StatusInternalServerError)
		return
	}
	return
})

var analysisHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	text, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Missing text to analyse.\n", http.StatusBadRequest)
		return
	}
	if len(text) > 0 {
		if strings.Compare(settings["speechtranslateto"], "off") != 0 {
			translatethis, err := gtranslate.TranslateWithParams(string(text), gtranslate.TranslationParams{From: settings["speechlanguage"], To: settings["speechtranslateto"]})
			if err != nil {
				fmt.Println(err)
			}
			if len(translatethis) > 0 {
				jsonOffer := map[string]interface{}{
					"Translation": translatethis,
				}
				json.NewEncoder(w).Encode(jsonOffer)
				return
			} else {
				jsonOffer := map[string]interface{}{
					"Translation": "google translate returns too many requests error please turn off the translate mode for a while with settings.txt",
				}
				json.NewEncoder(w).Encode(jsonOffer)
				return
			}
		} else {
			jsonOffer := map[string]interface{}{
				"Translation": "",
			}
			json.NewEncoder(w).Encode(jsonOffer)
			return
		}
	} else {
		http.Error(w, "Missing text.\n", http.StatusBadRequest)
	}
	return
})

func open(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

// Twitch
func task() {

	// AWS
	sess, errsess := session.NewSession(&aws.Config{
		Region:      aws.String("us-west-2"),
		Credentials: credentials.NewStaticCredentials(rootkeys["AWSAccessKeyId"], rootkeys["AWSSecretKey"], ""),
	})
	if errsess != nil {
		log.Fatal(errsess)
	}
	var svc = polly.New(sess)
	var com = comprehend.New(sess)

	// Twitch
	client := twitch.NewClient(settings["channel"], settings["oauth"])

	// Get Twitch Chat
	client.OnPrivateMessage(func(message twitch.PrivateMessage) {
		readthis = message.Message

		// Check Ignored Users
		for _, ignoreduser := range ignoredusers {
			if strings.Contains(message.User.Name, ignoreduser) == true {
				readthis = ""
				break
			}
		}

		// Read User Name
		if readthis != "" && strings.Compare(settings["readusername"], "on") == 0 {
			username := message.User.Name
			if strings.Compare(settings["ttsservice"], "aws") == 0 {
				username = "<speak>" + username + "</speak>"
				input = &polly.SynthesizeSpeechInput{OutputFormat: aws.String("mp3"), TextType: aws.String("ssml"), Text: aws.String(username), VoiceId: aws.String("Joanna")}
				output, err := svc.SynthesizeSpeech(input)
				if err != nil {
					if aerr, ok := err.(awserr.Error); ok {
						switch aerr.Code() {
						case polly.ErrCodeTextLengthExceededException:
							fmt.Println(polly.ErrCodeTextLengthExceededException, aerr.Error())
						case polly.ErrCodeInvalidSampleRateException:
							fmt.Println(polly.ErrCodeInvalidSampleRateException, aerr.Error())
						case polly.ErrCodeInvalidSsmlException:
							fmt.Println(polly.ErrCodeInvalidSsmlException, aerr.Error())
						case polly.ErrCodeLexiconNotFoundException:
							fmt.Println(polly.ErrCodeLexiconNotFoundException, aerr.Error())
						case polly.ErrCodeServiceFailureException:
							fmt.Println(polly.ErrCodeServiceFailureException, aerr.Error())
						case polly.ErrCodeMarksNotSupportedForFormatException:
							fmt.Println(polly.ErrCodeMarksNotSupportedForFormatException, aerr.Error())
						case polly.ErrCodeSsmlMarksNotSupportedForTextTypeException:
							fmt.Println(polly.ErrCodeSsmlMarksNotSupportedForTextTypeException, aerr.Error())
						case polly.ErrCodeLanguageNotSupportedException:
							fmt.Println(polly.ErrCodeLanguageNotSupportedException, aerr.Error())
						case polly.ErrCodeEngineNotSupportedException:
							fmt.Println(polly.ErrCodeEngineNotSupportedException, aerr.Error())
						default:
							fmt.Println(aerr.Error())
						}
					} else {
						fmt.Println(err.Error())
					}
					return
				}
				streamer, format, err := mp3.Decode(output.AudioStream)
				if err != nil {
					log.Fatal(err)
				}
				defer streamer.Close()
				speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
				done := make(chan bool)
				speaker.Play(beep.Seq(streamer, beep.Callback(func() {
					done <- true
				})))
				<-done
			} else if strings.Compare(settings["ttsservice"], "google-translate") == 0 {
				GoogleSpeak(username, "en")
			}
		}

		// Read Message
		if readthis != "" {

			// Check URL
			rxRelaxed := xurls.Relaxed()
			url := rxRelaxed.FindString(readthis)
			if url != "" {
				readthis = strings.ReplaceAll(readthis, url, "url")
			}

			// Check Emotes
			for _, emote := range message.Emotes {
				readthis = strings.ReplaceAll(readthis, emote.Name, "")
			}

			// Check Ignored Words
			for _, ignoredword := range ignoredwords {
				readthis = strings.ReplaceAll(readthis, ignoredword, "")
			}

			// Check RegEx
			for key, value := range regexs {
				re := regexp.MustCompile(key)
				readthis = re.ReplaceAllLiteralString(readthis, value)
			}

			if strings.Compare(settings["defaultlanguage"], "detect") == 0 {
				params := comprehend.DetectDominantLanguageInput{}
				params.SetText(TruncateString(readthis, 90))
				req, resp := com.DetectDominantLanguageRequest(&params)
				errreq := req.Send()
				if errreq == nil {
					lang = *resp.Languages[0].LanguageCode
				} else {
					fmt.Println(errreq)
				}
			} else if strings.Compare(settings["defaultlanguage"], "cpu") == 0 {
				info := detector1.Detect(readthis)
				lang = info.Lang.Iso6391()
				if info.Confidence < 0.8 {
					lang = detector2.Detect(strings.NewReader(readthis))
				}
			} else {
				lang = settings["defaultlanguage"]
			}

			if strings.Compare(settings["ttsservice"], "aws") == 0 {
				if strings.Compare(settings["chattranslateto"], "off") != 0 {
					readthis, err := gtranslate.TranslateWithParams(readthis, gtranslate.TranslationParams{From: lang, To: settings["chattranslateto"]})
					if err != nil {
						fmt.Println(err)
					}
					lang = settings["chattranslateto"]
					readthis = "<speak>" + settings["ssml-tag-start"] + readthis + settings["ssml-tag-end"] + "</speak>"
					if strings.Compare(lang, "ar") == 0 {
						input = &polly.SynthesizeSpeechInput{OutputFormat: aws.String("mp3"), TextType: aws.String("ssml"), Text: aws.String(readthis), VoiceId: aws.String(voices["Arabic"])}
					} else if strings.Compare(lang, "zh") == 0 {
						input = &polly.SynthesizeSpeechInput{OutputFormat: aws.String("mp3"), TextType: aws.String("ssml"), Text: aws.String(readthis), VoiceId: aws.String(voices["Chinese"])}
					} else if strings.Compare(lang, "da") == 0 {
						input = &polly.SynthesizeSpeechInput{OutputFormat: aws.String("mp3"), TextType: aws.String("ssml"), Text: aws.String(readthis), VoiceId: aws.String(voices["Danish"])}
					} else if strings.Compare(lang, "nl") == 0 {
						input = &polly.SynthesizeSpeechInput{OutputFormat: aws.String("mp3"), TextType: aws.String("ssml"), Text: aws.String(readthis), VoiceId: aws.String(voices["Dutch"])}
					} else if strings.Compare(lang, "en") == 0 {
						input = &polly.SynthesizeSpeechInput{OutputFormat: aws.String("mp3"), TextType: aws.String("ssml"), Text: aws.String(readthis), VoiceId: aws.String(voices["English"])}
					} else if strings.Compare(lang, "fr") == 0 {
						input = &polly.SynthesizeSpeechInput{OutputFormat: aws.String("mp3"), TextType: aws.String("ssml"), Text: aws.String(readthis), VoiceId: aws.String(voices["French"])}
					} else if strings.Compare(lang, "de") == 0 {
						input = &polly.SynthesizeSpeechInput{OutputFormat: aws.String("mp3"), TextType: aws.String("ssml"), Text: aws.String(readthis), VoiceId: aws.String(voices["German"])}
					} else if strings.Compare(lang, "hi") == 0 {
						input = &polly.SynthesizeSpeechInput{OutputFormat: aws.String("mp3"), TextType: aws.String("ssml"), Text: aws.String(readthis), VoiceId: aws.String(voices["Hindi"])}
					} else if strings.Compare(lang, "is") == 0 {
						input = &polly.SynthesizeSpeechInput{OutputFormat: aws.String("mp3"), TextType: aws.String("ssml"), Text: aws.String(readthis), VoiceId: aws.String(voices["Icelandic"])}
					} else if strings.Compare(lang, "it") == 0 {
						input = &polly.SynthesizeSpeechInput{OutputFormat: aws.String("mp3"), TextType: aws.String("ssml"), Text: aws.String(readthis), VoiceId: aws.String(voices["Italian"])}
					} else if strings.Compare(lang, "ja") == 0 {
						input = &polly.SynthesizeSpeechInput{OutputFormat: aws.String("mp3"), TextType: aws.String("ssml"), Text: aws.String(readthis), VoiceId: aws.String(voices["Japanese"])}
					} else if strings.Compare(lang, "ko") == 0 {
						input = &polly.SynthesizeSpeechInput{OutputFormat: aws.String("mp3"), TextType: aws.String("ssml"), Text: aws.String(readthis), VoiceId: aws.String(voices["Korean"])}
					} else if strings.Compare(lang, "no") == 0 {
						input = &polly.SynthesizeSpeechInput{OutputFormat: aws.String("mp3"), TextType: aws.String("ssml"), Text: aws.String(readthis), VoiceId: aws.String(voices["Norwegian"])}
					} else if strings.Compare(lang, "pl") == 0 {
						input = &polly.SynthesizeSpeechInput{OutputFormat: aws.String("mp3"), TextType: aws.String("ssml"), Text: aws.String(readthis), VoiceId: aws.String(voices["Polish"])}
					} else if strings.Compare(lang, "pt") == 0 {
						input = &polly.SynthesizeSpeechInput{OutputFormat: aws.String("mp3"), TextType: aws.String("ssml"), Text: aws.String(readthis), VoiceId: aws.String(voices["Portuguese"])}
					} else if strings.Compare(lang, "ro") == 0 {
						input = &polly.SynthesizeSpeechInput{OutputFormat: aws.String("mp3"), TextType: aws.String("ssml"), Text: aws.String(readthis), VoiceId: aws.String(voices["Romanian"])}
					} else if strings.Compare(lang, "ru") == 0 {
						input = &polly.SynthesizeSpeechInput{OutputFormat: aws.String("mp3"), TextType: aws.String("ssml"), Text: aws.String(readthis), VoiceId: aws.String(voices["Russian"])}
					} else if strings.Compare(lang, "es") == 0 {
						input = &polly.SynthesizeSpeechInput{OutputFormat: aws.String("mp3"), TextType: aws.String("ssml"), Text: aws.String(readthis), VoiceId: aws.String(voices["Spanish"])}
					} else if strings.Compare(lang, "sv") == 0 {
						input = &polly.SynthesizeSpeechInput{OutputFormat: aws.String("mp3"), TextType: aws.String("ssml"), Text: aws.String(readthis), VoiceId: aws.String(voices["Swedish"])}
					} else if strings.Compare(lang, "tr") == 0 {
						input = &polly.SynthesizeSpeechInput{OutputFormat: aws.String("mp3"), TextType: aws.String("ssml"), Text: aws.String(readthis), VoiceId: aws.String(voices["Turkish"])}
					} else if strings.Compare(lang, "cy") == 0 {
						input = &polly.SynthesizeSpeechInput{OutputFormat: aws.String("mp3"), TextType: aws.String("ssml"), Text: aws.String(readthis), VoiceId: aws.String(voices["Welsh"])}
					} else {
						input = &polly.SynthesizeSpeechInput{OutputFormat: aws.String("mp3"), TextType: aws.String("ssml"), Text: aws.String(readthis), VoiceId: aws.String(voices["Other"])}
					}
					output, err := svc.SynthesizeSpeech(input)
					if err != nil {
						if aerr, ok := err.(awserr.Error); ok {
							switch aerr.Code() {
							case polly.ErrCodeTextLengthExceededException:
								fmt.Println(polly.ErrCodeTextLengthExceededException, aerr.Error())
							case polly.ErrCodeInvalidSampleRateException:
								fmt.Println(polly.ErrCodeInvalidSampleRateException, aerr.Error())
							case polly.ErrCodeInvalidSsmlException:
								fmt.Println(polly.ErrCodeInvalidSsmlException, aerr.Error())
							case polly.ErrCodeLexiconNotFoundException:
								fmt.Println(polly.ErrCodeLexiconNotFoundException, aerr.Error())
							case polly.ErrCodeServiceFailureException:
								fmt.Println(polly.ErrCodeServiceFailureException, aerr.Error())
							case polly.ErrCodeMarksNotSupportedForFormatException:
								fmt.Println(polly.ErrCodeMarksNotSupportedForFormatException, aerr.Error())
							case polly.ErrCodeSsmlMarksNotSupportedForTextTypeException:
								fmt.Println(polly.ErrCodeSsmlMarksNotSupportedForTextTypeException, aerr.Error())
							case polly.ErrCodeLanguageNotSupportedException:
								fmt.Println(polly.ErrCodeLanguageNotSupportedException, aerr.Error())
							case polly.ErrCodeEngineNotSupportedException:
								fmt.Println(polly.ErrCodeEngineNotSupportedException, aerr.Error())
							default:
								fmt.Println(aerr.Error())
							}
						} else {
							fmt.Println(err.Error())
						}
						return
					}
					streamer, format, err := mp3.Decode(output.AudioStream)
					if err != nil {
						log.Fatal(err)
					}
					defer streamer.Close()
					speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
					done := make(chan bool)
					speaker.Play(beep.Seq(streamer, beep.Callback(func() {
						done <- true
					})))
					<-done
				} else {
					readthis = "<speak>" + settings["ssml-tag-start"] + readthis + settings["ssml-tag-end"] + "</speak>"
					if strings.Compare(lang, "ar") == 0 {
						input = &polly.SynthesizeSpeechInput{OutputFormat: aws.String("mp3"), TextType: aws.String("ssml"), Text: aws.String(readthis), VoiceId: aws.String(voices["Arabic"])}
					} else if strings.Compare(lang, "zh") == 0 {
						input = &polly.SynthesizeSpeechInput{OutputFormat: aws.String("mp3"), TextType: aws.String("ssml"), Text: aws.String(readthis), VoiceId: aws.String(voices["Chinese"])}
					} else if strings.Compare(lang, "da") == 0 {
						input = &polly.SynthesizeSpeechInput{OutputFormat: aws.String("mp3"), TextType: aws.String("ssml"), Text: aws.String(readthis), VoiceId: aws.String(voices["Danish"])}
					} else if strings.Compare(lang, "nl") == 0 {
						input = &polly.SynthesizeSpeechInput{OutputFormat: aws.String("mp3"), TextType: aws.String("ssml"), Text: aws.String(readthis), VoiceId: aws.String(voices["Dutch"])}
					} else if strings.Compare(lang, "en") == 0 {
						input = &polly.SynthesizeSpeechInput{OutputFormat: aws.String("mp3"), TextType: aws.String("ssml"), Text: aws.String(readthis), VoiceId: aws.String(voices["English"])}
					} else if strings.Compare(lang, "fr") == 0 {
						input = &polly.SynthesizeSpeechInput{OutputFormat: aws.String("mp3"), TextType: aws.String("ssml"), Text: aws.String(readthis), VoiceId: aws.String(voices["French"])}
					} else if strings.Compare(lang, "de") == 0 {
						input = &polly.SynthesizeSpeechInput{OutputFormat: aws.String("mp3"), TextType: aws.String("ssml"), Text: aws.String(readthis), VoiceId: aws.String(voices["German"])}
					} else if strings.Compare(lang, "hi") == 0 {
						input = &polly.SynthesizeSpeechInput{OutputFormat: aws.String("mp3"), TextType: aws.String("ssml"), Text: aws.String(readthis), VoiceId: aws.String(voices["Hindi"])}
					} else if strings.Compare(lang, "is") == 0 {
						input = &polly.SynthesizeSpeechInput{OutputFormat: aws.String("mp3"), TextType: aws.String("ssml"), Text: aws.String(readthis), VoiceId: aws.String(voices["Icelandic"])}
					} else if strings.Compare(lang, "it") == 0 {
						input = &polly.SynthesizeSpeechInput{OutputFormat: aws.String("mp3"), TextType: aws.String("ssml"), Text: aws.String(readthis), VoiceId: aws.String(voices["Italian"])}
					} else if strings.Compare(lang, "ja") == 0 {
						input = &polly.SynthesizeSpeechInput{OutputFormat: aws.String("mp3"), TextType: aws.String("ssml"), Text: aws.String(readthis), VoiceId: aws.String(voices["Japanese"])}
					} else if strings.Compare(lang, "ko") == 0 {
						input = &polly.SynthesizeSpeechInput{OutputFormat: aws.String("mp3"), TextType: aws.String("ssml"), Text: aws.String(readthis), VoiceId: aws.String(voices["Korean"])}
					} else if strings.Compare(lang, "no") == 0 {
						input = &polly.SynthesizeSpeechInput{OutputFormat: aws.String("mp3"), TextType: aws.String("ssml"), Text: aws.String(readthis), VoiceId: aws.String(voices["Norwegian"])}
					} else if strings.Compare(lang, "pl") == 0 {
						input = &polly.SynthesizeSpeechInput{OutputFormat: aws.String("mp3"), TextType: aws.String("ssml"), Text: aws.String(readthis), VoiceId: aws.String(voices["Polish"])}
					} else if strings.Compare(lang, "pt") == 0 {
						input = &polly.SynthesizeSpeechInput{OutputFormat: aws.String("mp3"), TextType: aws.String("ssml"), Text: aws.String(readthis), VoiceId: aws.String(voices["Portuguese"])}
					} else if strings.Compare(lang, "ro") == 0 {
						input = &polly.SynthesizeSpeechInput{OutputFormat: aws.String("mp3"), TextType: aws.String("ssml"), Text: aws.String(readthis), VoiceId: aws.String(voices["Romanian"])}
					} else if strings.Compare(lang, "ru") == 0 {
						input = &polly.SynthesizeSpeechInput{OutputFormat: aws.String("mp3"), TextType: aws.String("ssml"), Text: aws.String(readthis), VoiceId: aws.String(voices["Russian"])}
					} else if strings.Compare(lang, "es") == 0 {
						input = &polly.SynthesizeSpeechInput{OutputFormat: aws.String("mp3"), TextType: aws.String("ssml"), Text: aws.String(readthis), VoiceId: aws.String(voices["Spanish"])}
					} else if strings.Compare(lang, "sv") == 0 {
						input = &polly.SynthesizeSpeechInput{OutputFormat: aws.String("mp3"), TextType: aws.String("ssml"), Text: aws.String(readthis), VoiceId: aws.String(voices["Swedish"])}
					} else if strings.Compare(lang, "tr") == 0 {
						input = &polly.SynthesizeSpeechInput{OutputFormat: aws.String("mp3"), TextType: aws.String("ssml"), Text: aws.String(readthis), VoiceId: aws.String(voices["Turkish"])}
					} else if strings.Compare(lang, "cy") == 0 {
						input = &polly.SynthesizeSpeechInput{OutputFormat: aws.String("mp3"), TextType: aws.String("ssml"), Text: aws.String(readthis), VoiceId: aws.String(voices["Welsh"])}
					} else {
						input = &polly.SynthesizeSpeechInput{OutputFormat: aws.String("mp3"), TextType: aws.String("ssml"), Text: aws.String(readthis), VoiceId: aws.String(voices["Other"])}
					}
					output, err := svc.SynthesizeSpeech(input)
					if err != nil {
						if aerr, ok := err.(awserr.Error); ok {
							switch aerr.Code() {
							case polly.ErrCodeTextLengthExceededException:
								fmt.Println(polly.ErrCodeTextLengthExceededException, aerr.Error())
							case polly.ErrCodeInvalidSampleRateException:
								fmt.Println(polly.ErrCodeInvalidSampleRateException, aerr.Error())
							case polly.ErrCodeInvalidSsmlException:
								fmt.Println(polly.ErrCodeInvalidSsmlException, aerr.Error())
							case polly.ErrCodeLexiconNotFoundException:
								fmt.Println(polly.ErrCodeLexiconNotFoundException, aerr.Error())
							case polly.ErrCodeServiceFailureException:
								fmt.Println(polly.ErrCodeServiceFailureException, aerr.Error())
							case polly.ErrCodeMarksNotSupportedForFormatException:
								fmt.Println(polly.ErrCodeMarksNotSupportedForFormatException, aerr.Error())
							case polly.ErrCodeSsmlMarksNotSupportedForTextTypeException:
								fmt.Println(polly.ErrCodeSsmlMarksNotSupportedForTextTypeException, aerr.Error())
							case polly.ErrCodeLanguageNotSupportedException:
								fmt.Println(polly.ErrCodeLanguageNotSupportedException, aerr.Error())
							case polly.ErrCodeEngineNotSupportedException:
								fmt.Println(polly.ErrCodeEngineNotSupportedException, aerr.Error())
							default:
								fmt.Println(aerr.Error())
							}
						} else {
							fmt.Println(err.Error())
						}
						return
					}
					streamer, format, err := mp3.Decode(output.AudioStream)
					if err != nil {
						log.Fatal(err)
					}
					defer streamer.Close()
					speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
					done := make(chan bool)
					speaker.Play(beep.Seq(streamer, beep.Callback(func() {
						done <- true
					})))
					<-done
				}
			} else if strings.Compare(settings["ttsservice"], "google-translate") == 0 {
				if strings.Compare(settings["chattranslateto"], "off") != 0 {
					readthistranslated, err := gtranslate.TranslateWithParams(readthis, gtranslate.TranslationParams{From: lang, To: settings["chattranslateto"]})
					if err != nil {
						fmt.Println(err)
						GoogleSpeak("google translate returns too many requests error please turn off the translate mode for a while with settings.txt", "en")
					}
					if len(readthistranslated) > 0 {
						lang = settings["chattranslateto"]
						GoogleSpeak(readthistranslated, lang)
					} else {
						GoogleSpeak(readthis, lang)
					}
				} else {
					GoogleSpeak(readthis, lang)
				}
			}
		}
	})

	// Twitch Join Channel
	client.Join(settings["channel"])
	errclient := client.Connect()
	if errclient != nil {
		panic(errclient)
	}
}

// Discord
func discord(token string) {
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}
	dg.AddHandler(messageCreate)
	dg.Identify.Intents = discordgo.IntentsGuildMessages
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}
}

// Get Discord Chat
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	dreadthis := m.Content
	dlang := "en"
	dinfo := detector1.Detect(dreadthis)
	dlang = dinfo.Lang.Iso6391()
	if dinfo.Confidence < 0.8 {
		dlang = detector2.Detect(strings.NewReader(dreadthis))
		GoogleSpeak(dreadthis, dlang)
	} else {
		GoogleSpeak(dreadthis, dlang)
	}
}
