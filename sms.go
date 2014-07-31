package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
)

type Message struct {
	From string `json:"from"`
	To   string `json:"to"`
	Text string `json:"text"`
}
type SMSPilotMessage struct {
	Send   []Message `json:"send"`
	ApiKey string    `json:"apikey"`
}
type PhoneType struct {
	Number string
	Name   string
}

func sendPush(phone, text string) {
	apiUrl := "https://api.pushover.net/1/messages.json"
	data := url.Values{}
	data.Set("token", "h6RToHDU7gNnB3IMyUb94SuwKtBzOD")
	data.Add("user", phone)
	data.Add("sound", "siren")
	data.Add("message", text)
	u, _ := url.ParseRequestURI(apiUrl)
	u.RawQuery = data.Encode()
	urlStr := fmt.Sprintf("%v", u)
	client := &http.Client{}
	r, _ := http.NewRequest("POST", urlStr, nil)
	resp, err := client.Do(r)
	if err != nil {
		log.Println("Unable to sent push notification")
		return
	}
	defer resp.Body.Close()
	//body, err := ioutil.ReadAll(resp.Body)
	//if err == nil {
	//	fmt.Println(string(body))
	//}
}

func sendSMS(phone string, text string) {
	//settings, _ := ("hget", "gopoller/smspilot").Hash()
	settings := LoadSMSSettings()

	apiURL := "http://smspilot.ru/api2.php"
	buffer := make([]Message, 1)
	message := Message{
		From: settings.From,
		To:   phone,
		Text: text}
	buffer[0] = message
	if len(settings.ApiKey) > 10 {
		sms := &SMSPilotMessage{Send: buffer, ApiKey: settings.ApiKey}
		b, _ := json.Marshal(sms)
		fmt.Println(string(b))
		bb := strings.NewReader(string(b))
		resp, _ := http.Post(apiURL, "application/json", bb)
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err == nil {
			log.Print(string(body))
		}
	}
}
