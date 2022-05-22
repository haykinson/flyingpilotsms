package main

import (
	"errors"
	"fmt"
	"github.com/twilio/twilio-go"
	"log"
	"net/http"
	"os"
	"strings"
	"unicode"
)

type metar struct {
	Error     string `json:"error"`
	Sanitized string `json:"sanitized"`
}

type taf struct {
	Error string `json:"error"`
	Raw   string `json:"raw"`
}

type command struct {
	code     string
	getMetar bool
	getTaf   bool
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

	log.Println(fmt.Sprintf("Incoming message: %v", body))

	if !onlyValidChars(body) {
		log.Println(fmt.Sprintf("Incoming body has invalid chars: %v", body))
		return
	}

	ctx := &context{twilioClient: twilioClient, sender: sender, destination: from}
	go handleMessage(ctx, body)
}

func onlyValidChars(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

func handleMessage(w weatherHandler, message string) {
	command, err := getCommandFromMessage(message)
	if err != nil {
		log.Println(fmt.Sprintf("Problem parsing message: %v", err))
		return
	}

	var metar *metar
	var taf *taf

	if command.getMetar {
		metar, err = w.handleMetar(command.code)
		if err != nil {
			log.Println("Asked to get metar, but couldn't get it")
		} else {
			fmt.Println(metar.Sanitized)
		}
	}

	if command.getTaf {
		taf, err = w.handleTaf(command.code)
		if err != nil {
			log.Println("Asked to get taf, but couldn't get it")
		} else {
			fmt.Println(taf.Raw)
		}
	}

	if taf == nil && metar == nil {
		return
	}

	response := ""

	if metar != nil {
		response += metar.Sanitized
	}

	if taf != nil {
		if len(response) > 0 {
			response += "\n\n"
		}
		response += taf.Raw
	}

	w.sendMessage(response)
}

func getCommandFromMessage(message string) (*command, error) {
	normalizedMessage := strings.ToUpper(strings.TrimSpace(message))

	if len(normalizedMessage) < 3 {
		return nil, errors.New("not enough text in command")
	}

	segments := strings.Split(normalizedMessage, " ")
	removeEmpty(&segments)

	var r command
	for _, item := range segments {
		switch strings.ToUpper(item) {
		case "METAR":
			r.getMetar = true
		case "TAF":
			r.getTaf = true
		default:
			r.code = item
		}
	}

	if !r.getMetar && !r.getTaf {
		r.getMetar = true
		r.getTaf = true
	}

	return &r, nil
}

func removeEmpty(segments *[]string) {
	var result []string
	for _, item := range *segments {
		trimmed := strings.TrimSpace(item)
		if len(trimmed) > 0 {
			result = append(result, item)
		}
	}
	*segments = result
}
