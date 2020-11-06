/*
history:
2017-0520 v1
2017-0320 v2
2019-0927 TrimSpace(message)
2020-1026 renamed to teleji to add audio and video messages support
20/1106 print message id of the posted message and able to edit messages by id

https://core.telegram.org/bots/api

GoFmt GoBuild GoRelease GoRun
*/

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	TgToken     string
	TgChatIds   []int
	TgMessageId int
	CgiMode     bool
)

func log(msg string, args ...interface{}) {
	const Beat = time.Duration(24) * time.Hour / 1000
	tzBiel := time.FixedZone("Biel", 60*60)
	t := time.Now().In(tzBiel)
	ty := t.Sub(time.Date(t.Year(), 1, 1, 0, 0, 0, 0, tzBiel))
	td := t.Sub(time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, tzBiel))
	ts := fmt.Sprintf(
		"%d/%d@%d",
		t.Year()%1000,
		int(ty/(time.Duration(24)*time.Hour))+1,
		int(td/Beat),
	)
	fmt.Fprintf(os.Stderr, ts+" "+msg+"\n", args...)
}

func main() {
	var err error

	TgToken := os.Getenv("TgToken")
	if TgToken == "" {
		log("Empty TgToken env var.")
		os.Exit(1)
	}

	for _, i := range strings.Split(os.Getenv("TgChatId"), ",") {
		if i == "" {
			continue
		}
		chatid, err := strconv.Atoi(i)
		if err != nil || chatid == 0 {
			log("Invalid chat id `%s`", i)
		}
		TgChatIds = append(TgChatIds, chatid)
	}
	if len(TgChatIds) == 0 {
		log("Empty or invalid TgChatId env var.")
		os.Exit(1)
	}

	if os.Getenv("TgMessageId") != "" {
		if len(TgChatIds) > 1 {
			log("Edit message by id only with one chat id")
			os.Exit(1)
		}
		TgMessageId, err = strconv.Atoi(os.Getenv("TgMessageId"))
		if err != nil || TgMessageId == 0 {
			log("invalid message id `%s`", os.Getenv("TgMessageId"))
			os.Exit(1)
		}
	}

	messageBytes, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log("%v", err)
		os.Exit(1)
	}
	message := strings.TrimSpace(string(messageBytes))
	if message == "" {
		log("Empty message.")
		os.Exit(1)
	}

	if len(os.Args) > 1 && os.Args[1] == "-cgi" {
		CgiMode = true
	}

	//remotehost := os.Getenv("REMOTE_HOST")
	//if CgiMode {
	//	message = fmt.Sprintf("%s/ %s", remotehost, message)
	//}

	tgprefix := os.Getenv("TgPrefix")
	if tgprefix != "" {
		message = fmt.Sprintf("%s%s", tgprefix, message)
	}
	tgsuffix := os.Getenv("TgSuffix")
	if tgsuffix != "" {
		message = fmt.Sprintf("%s%s", message, tgsuffix)
	}

	disablenotification := true
	parsemode := "Markdown"

	var smresp struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
		Result      struct {
			MessageId int64 `json:"message_id"`
		} `json:"result"`
	}

	if TgMessageId == 0 {
		for _, chatid := range TgChatIds {
			sendMessage := map[string]interface{}{
				"chat_id":              chatid,
				"text":                 message,
				"disable_notification": disablenotification,
				"parse_mode":           parsemode,
			}
			sendMessageJSON, err := json.Marshal(sendMessage)
			if err != nil {
				log("%v", err)
				os.Exit(1)
			}
			sendMessageJSONBuffer := bytes.NewBuffer(sendMessageJSON)

			resp, err := http.Post(
				fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", TgToken),
				"application/json",
				sendMessageJSONBuffer,
			)
			if err != nil {
				log("%v", err)
				os.Exit(1)
			}

			err = json.NewDecoder(resp.Body).Decode(&smresp)
			if err != nil {
				log("%v", err)
				os.Exit(1)
			}
			if !smresp.OK {
				log("sendMessage was not OK: %+v", smresp)
				os.Exit(1)
			}
		}
	} else {
		editMessageText := map[string]interface{}{
			"chat_id":              TgChatIds[0],
			"message_id":           TgMessageId,
			"text":                 message,
			"disable_notification": disablenotification,
			"parse_mode":           parsemode,
		}
		editMessageTextJSON, err := json.Marshal(editMessageText)
		if err != nil {
			log("%v", err)
			os.Exit(1)
		}
		editMessageTextJSONBuffer := bytes.NewBuffer(editMessageTextJSON)

		resp, err := http.Post(
			fmt.Sprintf("https://api.telegram.org/bot%s/editMessageText", TgToken),
			"application/json",
			editMessageTextJSONBuffer,
		)
		if err != nil {
			log("%v", err)
			os.Exit(1)
		}

		err = json.NewDecoder(resp.Body).Decode(&smresp)
		if err != nil {
			log("%v", err)
			os.Exit(1)
		}
		if !smresp.OK {
			log("editMessageText was not OK: %+v", smresp)
			os.Exit(1)
		}
	}

	if CgiMode {
		fmt.Println("Content-Type: text/plain")
		fmt.Println("Content-Length: 0")
		fmt.Println()
	}

	fmt.Printf("%d\n", smresp.Result.MessageId)
}
