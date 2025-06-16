/*
history:
2017-0520 v1
2017-0320 v2
2019-0927 TrimSpace(message)
2020-1026 renamed to teleji to add audio and video messages support
020/1106 print message id of the posted message and able to edit messages by id
020/1118 TgPre env var to send a preformatted message
021/0916 TgDisableWebPagePreview
024/0529 TgMessageText env var instead of reading from stdin
024/0529 escape cmd

usage:
teleji - reads text from TgMessageText env var and sends the message
teleji escape - reads text from TgMessageText env var, prints escaped text to stdout
teleji escape VAR_NAME - reads text from VAR_NAME env var, prints escaped text to stdout
teleji version - prints version to stdout

https://core.telegram.org/bots/api

GoGet GoFmt GoBuildNull GoBuild
GoRun

TODO escape in main
*/

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	Version string
	Verbose bool

	TgToken                 string
	TgChatIds               []int64
	TgMessageIds            []int
	TgPrefix                string
	TgSuffix                string
	TgParseMode             string
	TgDisableNotification   bool = true
	TgDisableWebPagePreview bool = true
	TgPre                   bool = false
	//CgiMode     bool

	TgMessageText string
)

const (
	NL = "\n"
)

func init() {

	if len(os.Args) == 2 && (os.Args[1] == "version" || os.Args[1] == "--version") {
		fmt.Println(Version)
		os.Exit(0)
	}

	TgMessageText = os.Getenv("TgMessageText")
	TgMessageText = strings.TrimSpace(TgMessageText)

	if len(os.Args) > 1 && os.Args[1] == "escape" {

		if len(os.Args) > 2 {
			TgMessageText = os.Getenv(os.Args[2])
			TgMessageText = strings.TrimSpace(TgMessageText)
		}

		// https://core.telegram.org/bots/api#markdownv2-style
		for _, c := range "\\_*[]()~`>#+-=|{}.!" {
			TgMessageText = strings.ReplaceAll(TgMessageText, string(c), "\\"+string(c))
		}

		fmt.Println(TgMessageText)
		os.Exit(0)

	}

	if TgMessageText == "" {
		log("Empty TgMessageText.")
		os.Exit(1)
	}

	if os.Getenv("Verbose") != "" {
		Verbose = true
	}

	TgToken = os.Getenv("TgToken")
	if TgToken == "" {
		log("Empty TgToken env var.")
		os.Exit(1)
	}

	for _, i := range strings.Split(os.Getenv("TgChatId"), ",") {
		if i == "" {
			continue
		}
		var chatid int64
		var err error
		chatid, err = strconv.ParseInt(i, 10, 64)
		if err != nil || chatid == 0 {
			log("Invalid chat id `%s`", i)
		}
		TgChatIds = append(TgChatIds, chatid)
	}
	if len(TgChatIds) == 0 {
		log("Empty or invalid TgChatId env var.")
		os.Exit(1)
	}

	for _, i := range strings.Split(os.Getenv("TgMessageId"), ",") {
		if i == "" {
			continue
		}
		messageid, err := strconv.Atoi(i)
		if err != nil || messageid == 0 {
			log("Invalid message id `%s`", i)
		}
		TgMessageIds = append(TgMessageIds, messageid)
	}
	if len(TgMessageIds) > 0 && len(TgMessageIds) != len(TgChatIds) {
		log("Number of message ids should be equal to number of chat ids.")
		os.Exit(1)
	}

	TgParseMode = os.Getenv("TgParseMode")
	TgPrefix = os.Getenv("TgPrefix")
	TgSuffix = os.Getenv("TgSuffix")

	if os.Getenv("TgPre") != "" {
		TgPre = true
	}
}

func main() {
	var err error

	/*
		if len(os.Args) > 1 && os.Args[1] == "-cgi" {
			CgiMode = true
		}

		remotehost := os.Getenv("REMOTE_HOST")
		if CgiMode {
			message = fmt.Sprintf("%s/ %s", remotehost, message)
		}
	*/

	if TgPrefix != "" {
		TgMessageText = TgPrefix + TgMessageText
	}
	if TgSuffix != "" {
		TgMessageText = TgMessageText + TgSuffix
	}

	if TgPre {
		for _, c := range "\\`" {
			TgMessageText = strings.ReplaceAll(TgMessageText, string(c), "\\"+string(c))
		}
		TgMessageText = "```" + NL + TgMessageText + NL + "```"
		TgParseMode = "MarkdownV2"
	}

	var smresp struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
		Result      struct {
			MessageId int64 `json:"message_id"`
		} `json:"result"`
	}
	var resp *http.Response

	for i, _ := range TgChatIds {
		if len(TgMessageIds) == 0 {
			sendMessage := TgSendMessageRequest{
				ChatId:                TgChatIds[i],
				Text:                  TgMessageText,
				ParseMode:             TgParseMode,
				DisableNotification:   TgDisableNotification,
				DisableWebPagePreview: TgDisableWebPagePreview,
			}
			sendMessageJSON, err := json.Marshal(sendMessage)
			if err != nil {
				log("%v", err)
				os.Exit(1)
			}
			sendMessageJSONBuffer := bytes.NewBuffer(sendMessageJSON)
			if Verbose {
				log("json: %v", sendMessageJSONBuffer)
			}

			requrl := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", TgToken)
			if Verbose {
				log("url: %v", requrl)
			}
			resp, err = http.Post(
				requrl,
				"application/json",
				sendMessageJSONBuffer,
			)
			if err != nil {
				log("%v", err)
				os.Exit(1)
			}
		} else {
			editMessageText := TgEditMessageRequest{
				TgSendMessageRequest: TgSendMessageRequest{
					ChatId:                TgChatIds[i],
					Text:                  TgMessageText,
					ParseMode:             TgParseMode,
					DisableNotification:   TgDisableNotification,
					DisableWebPagePreview: TgDisableWebPagePreview,
				},
				MessageId: int64(TgMessageIds[i]),
			}
			editMessageTextJSON, err := json.Marshal(editMessageText)
			if err != nil {
				log("%v", err)
				os.Exit(1)
			}
			editMessageTextJSONBuffer := bytes.NewBuffer(editMessageTextJSON)
			if Verbose {
				log("json: %v", editMessageTextJSONBuffer)
			}

			requrl := fmt.Sprintf("https://api.telegram.org/bot%s/editMessageText", TgToken)
			if Verbose {
				log("url: %v", requrl)
			}
			resp, err = http.Post(
				requrl,
				"application/json",
				editMessageTextJSONBuffer,
			)
			if err != nil {
				log("%v", err)
				os.Exit(1)
			}
		}

		if Verbose {
			log("resp.StatusCode: %v", resp.StatusCode)
		}
		err = json.NewDecoder(resp.Body).Decode(&smresp)
		if err != nil {
			log("%v", err)
			os.Exit(1)
		}
		if !smresp.OK {
			log("Api response not OK: %+v", smresp)
			os.Exit(1)
		}

		fmt.Printf("%d\n", smresp.Result.MessageId)
	}

	/*
		if CgiMode {
			fmt.Println("Content-Type: text/plain")
			fmt.Println("Content-Length: 0")
			fmt.Println()
		}
	*/
}

func ts() string {
	tnow := time.Now().Local()
	return fmt.Sprintf(
		"%d%02d%02d:%02d%02d",
		tnow.Year()%1000, tnow.Month(), tnow.Day(),
		tnow.Hour(), tnow.Minute(),
	)
}

func log(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, ts()+" "+msg+NL, args...)
}

type TgSendMessageRequest struct {
	ChatId                int64  `json:"chat_id"`
	Text                  string `json:"text"`
	ParseMode             string `json:"parse_mode,omitempty"`
	DisableNotification   bool   `json:"disable_notification"`
	DisableWebPagePreview bool   `json:"disable_web_page_preview"`
}

type TgEditMessageRequest struct {
	TgSendMessageRequest
	MessageId int64 `json:"message_id"`
}
