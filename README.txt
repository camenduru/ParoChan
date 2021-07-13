Twitch Parrot (ParoChan) can recognize speech, detect, translate and read languages: (system tray app​) (speech-to-text and text-to speech and translator)

[Arabic], [Chinese, Mandarin], [Danish], [Dutch], [English], [French], [German], [Hindi], [Icelandic], [Italian], [Japanese], [Korean], [Norwegian], [Polish], [Portuguese], [Romanian], [Russian], [Spanish], [Swedish], [Turkish], [Welsh]

Twitch Parrot (ParoChan) Uses Amazon Comprehend or CPU for Language Detection and Amazon Polly or Google Translate for Text To Speech and Google Translate for Translate and Google Chrome for Speech recognition

(If you are planning not to use Amazon Comprehend or Amazon Polly you don't need setup this part)
For Amazon Credential
go to https://console.aws.amazon.com/iam/home#/security_credential 
(if haven't registered user then to have sign up) and open Access Keys tab
Create New Access Key
Download Key File
you will get rootkey.csv put it in TwitchParrot Folder

For Twitch Credential
enter channel name in settings.txt
enter your oauth in settings.txt - You can generate one at https://twitchapps.com/tmi

Twitch Parrot (ParoChan) has no auto update sorry! go to https://camenduru.itch.io/twitchparrot and check latest version

for more dev https://www.twitch.tv/camenduru

Check AWS Free Tiers
https://aws.amazon.com/polly/pricing/
https://aws.amazon.com/comprehend/pricing/

if you have idea write in the comment in https://camenduru.itch.io/twitchparrot

--- settings cheat sheet ---

[ignoredusers.txt     ] [enter ignored users per line one user] default empty
[ignoredwords.txt     ] [enter ignored words per line one word] default empty
[regexlist.txt        ] [find and replace per line one regex] default empty
[voicelist.txt        ] [amazon polly voice selection for each language]
[rootkey.csv          ] [create and download from https://console.aws.amazon.com/iam/home#/security_credential]
[speech.html          ] [you can edit text style]

[settings.txt]
[channel              ] [enter your channel name] default empty
[oauth                ] [enter your oauth from https://twitchapps.com/tmi] default empty
[readusername         ] [on][off] default[off]
[defaultlanguage      ] [detect][cpu][ar][zh][da][nl][en][fr][de][hi][is][it][ja][ko][no][pl][pt][ro][ru][es][sv][tr][cy] default[cpu]
[speechlanguage       ] [off][ar][zh][da][nl][en][fr][de][hi][is][it][ja][ko][no][pl][pt][ro][ru][es][sv][tr][cy] default[off]
[ttsservice           ] [aws][google-translate] default[google-translate] 
[chattranslateto      ] [off][ar][zh][da][nl][en][fr][de][hi][is][it][ja][ko][no][pl][pt][ro][ru][es][sv][tr][cy] default[off]
[speechtranslateto    ] [off][ar][zh][da][nl][en][fr][de][hi][is][it][ja][ko][no][pl][pt][ro][ru][es][sv][tr][cy] default[off]

--- changelog ---

v0.2
- ignores emotes

v0.3
- ignores url
- ignores user with ignoredusers.txt

v0.4
- ignores users with ignoredusers.txt (use lowercase usernames, each line one username)
- ignores words with ignoredwords.txt (each line one word)

v0.5
- amazon comprehend usage cost lowered
- read display name function removed (detecting language of display name costly)
- read user name [on] [off] with readusername (reads username only english speaker) default [off]

v0.6
- detecting language or default language option [detect] [cpu] [ar] [zh] [da] [nl] [en] [fr] [de] [hi] [is] [it] [ja] [ko] [no] [pl] [pt] [ro] [ru] [es] [sv] [tr] [cy] with defaultlanguage.txt default [detect]
( if you use detect option you will use Amazon Comprehend 50K chat line free per month with Amazon Comprehend if you are using with one channel it is ok but if you are using Twitch Parrot with multiple channel use default language like ja for Japanese using default channel disables Amazon Comprehend usage now you are only spend your Amazon Polly Free Tier (First 5,000,000 characters per month are free) check your AWS Free Tier Summary with this link https://console.aws.amazon.com/billing/home?nc2=h_m_bc#/freetier )

v0.7
- detecting language with cpu [detect] [cpu] [ar] [zh] [da] [nl] [en] [fr] [de] [hi] [is] [it] [ja] [ko] [no] [pl] [pt] [ro] [ru] [es] [sv] [tr] [cy] with defaultlanguage default [cpu]
( this method can not detect all languages not accurate all the time but it is something )

v0.8
- voice selection with voicelist.txt (if name has fancy character like Céline use Celine if Joanna (Standard) use only Joanna you can find example of voices for each language from https://aws.amazon.com/polly/features idea from twitch.tv/mashedkiwi )
- regular expression find and replace with regexlist.txt (each line one regex ~ sign and word like ( [8８]{4,}~ぱちぱち this changes equal and more then 8 and ８ to ぱちぱち) and ( [@]~ this stops reading before mention @ sign )  idea from twitch.tv/hskwakr )

v0.9
- tts service selection with [aws] [google-translate] ttsservice default [google-translate] (if you don't have AWS account you can use google-translate option for tts service provider not better then Amazon Polly also you can not use detect option with google-translate but it is something and free )

v1.0
- google translate [off] [ar] [zh] [da] [nl] [en] [fr] [de] [hi] [is] [it] [ja] [ko] [no] [pl] [pt] [ro] [ru] [es] [sv] [tr] [cy] chattranslateto default [off] (free)

v1.1
- speech recognition and google translate [off] [ar] [zh] [da] [nl] [en] [fr] [de] [hi] [is] [it] [ja] [ko] [no] [pl] [pt] [ro] [ru] [es] [sv] [tr] [cy] speechlanguage default [off] speechtranslateto default [off] (free)
(if you want to use this function you should use google chrome latest version and you should use obs chroma key filter with the opened page's green background and crop the top of the page in obs this chrome tab will be recognize your speech and then translate it should be only one Twitch Parrot chrome tab also you can edit text style with speech.html)

v1.2
- bug fix (if google translate returns too many requests error please turn off the translate mode for a while with chattranslateto and speechtranslateto)

v1.3
- speech recognition improvement

v1.4
- some config txt files ​simplified to settings.txt
- open config txt files with menu if any changes in config files auto close the Twitch Parrot (ParoChan) why? because you should restart when you update any config txt file

v1.5
- settings.txt and regexlist.txt separator character changed from = to ~ (ssml tags contains =)
- amazon polly ssml tag support (https://docs.aws.amazon.com/polly/latest/dg/supportedtags.html) with ssml-tag-start and ssml-tag-end in settings.txt or you can use directly in chat default empty thanks for the idea twitch.tv/mashedkiwi
(you can use this feature with regexlist you can create your own command for ssml tag like if you add w:~<amazon:effect name="whispered"> and :w~</amazon:effect> to your regexlist.txt polly will start whispering when someone writes whisper: this is secret :whisper)

v1.6
- discord support (experimental) create your discord bot give it a read and write permission add your token to discord-token ~ yourdiscordbottoken in settings.txt