package main

import (
	"encoding/json"
	"fmt"
	"github.com/twilio/twilio-go"
	openapi "github.com/twilio/twilio-go/rest/api/v2010"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"unicode"
)

type Metar struct {
	Sanitized string `json:"sanitized"`
}

var avwxToken = os.Getenv("AVWX_TOKEN")
var sender = os.Getenv("TWILIO_PHONE_NUMBER")
var twilioClient = twilio.NewRestClient()

func main() {
	http.HandleFunc("/incoming_sms", processIncoming)
	log.Fatal(http.ListenAndServe(":8123", nil))
}

func processIncoming(_ http.ResponseWriter, r *http.Request) {
	messageSid := r.FormValue("MessageSid")

	if len(messageSid) == 0 {
		log.Println("Incoming request had no message sid")
		return
	}

	from := r.FormValue("From")

	if len(from) == 0 {
		log.Println("Incoming request had no from number")
		return
	}

	rawBody := r.FormValue("Body")
	if len(rawBody) == 0 {
		log.Println("Incoming request had no body")
		return
	}

	body := strings.TrimSpace(rawBody)

	if len(body) < 3 || len(body) > 4 || !isAllLettersAndNumbers(body) {
		log.Println(fmt.Sprintf("Incoming body not four chars or not all letters, probably not an airport: %v", body))
		return
	}

	go handleMetar(from, body)
}

func isAllLettersAndNumbers(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func handleMetar(from string, icao string) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", fmt.Sprintf("https://avwx.rest/api/metar/%v?options=&airport=true&reporting=true&format=json&remove=&filter=sanitized&onfail=cache", icao), nil)

	req.Header.Add("Authorization", avwxToken)
	resp, err := client.Do(req)

	if err != nil {
		log.Println(err)
		return
	}

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Println(err)
		return
	}

	metar := &Metar{}

	err = json.Unmarshal(body, &metar)

	if err != nil {
		log.Println(err)
		return
	}

	fmt.Println(metar.Sanitized)

	sendMessage(twilioClient, metar.Sanitized, sender, from)
}

func sendMessage(client *twilio.RestClient, message string, from string, to string) {
	params := &openapi.CreateMessageParams{}
	params.SetTo(to)
	params.SetFrom(from)
	params.SetBody(message)

	resp, err := client.Api.CreateMessage(params)
	if err != nil {
		fmt.Println(err.Error())
	} else {
		response, _ := json.Marshal(*resp)
		fmt.Println("Response: " + string(response))
	}

}