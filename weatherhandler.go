package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/twilio/twilio-go"
	"github.com/twilio/twilio-go/rest/api/v2010"
	"io"
	"io/ioutil"
	"log"
	"net/http"
)

type context struct {
	twilioClient *twilio.RestClient
	sender       string
	destination  string
}

type weatherHandler interface {
	handleMetar(code string) (*metar, error)
	handleTaf(code string) (*taf, error)
	sendMessage(message string)
}

func (c *context) handleMetar(code string) (*metar, error) {
	metar, fail := fetchMetar(code)
	if fail {
		return nil, errors.New("could not fetch metar")
	}

	if len(metar.Error) > 0 {
		if len(code) != 3 {
			log.Println(fmt.Sprintf("Error in metar for %v: %v", code, metar.Error))
			return nil, errors.New("not found")
		}

		// try again by prepending a "K"
		prependedCode := "K" + code
		metar, fail = fetchMetar(prependedCode)
		if fail {
			return nil, errors.New("could not fetch metar")
		}

		if len(metar.Error) > 0 {
			log.Println(fmt.Sprintf("Error in metar for %v: %v", prependedCode, metar.Error))
			return nil, errors.New("not found after prepending K to shorter code")
		}
	}

	return metar, nil
}

func (c *context) handleTaf(code string) (*taf, error) {
	taf, fail := fetchTaf(code)
	if fail {
		return nil, errors.New("could not fetch taf")
	}

	if len(taf.Error) > 0 {
		if len(code) != 3 {
			log.Println(fmt.Sprintf("Error in taf for %v: %v", code, taf.Error))
			return nil, errors.New("not found")
		}

		// try again by prepending a "K"
		prependedCode := "K" + code
		taf, fail = fetchTaf(prependedCode)
		if fail {
			return nil, errors.New("could not fetch taf")
		}

		if len(taf.Error) > 0 {
			log.Println(fmt.Sprintf("Error in taf for %v: %v", prependedCode, taf.Error))
			return nil, errors.New("not found after prepending K to shorter code")
		}
	}

	return taf, nil
}

func fetchMetar(icao string) (*metar, bool) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", fmt.Sprintf("https://avwx.rest/api/metar/%v?options=&airport=true&reporting=true&format=json&remove=&filter=sanitized&onfail=cache", icao), nil)

	req.Header.Add("Authorization", avwxToken)
	resp, err := client.Do(req)

	if err != nil {
		log.Println(err)
		return nil, true
	}

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Println(err)
		return nil, true
	}

	metar := &metar{}

	err = json.Unmarshal(body, &metar)

	if err != nil {
		log.Println(err)
		return nil, true
	}
	return metar, false
}

func fetchTaf(code string) (*taf, bool) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", fmt.Sprintf("https://avwx.rest/api/taf/%v?options=&airport=true&reporting=true&format=json&remove=&filter=raw&onfail=cache", code), nil)

	req.Header.Add("Authorization", avwxToken)
	resp, err := client.Do(req)

	if err != nil {
		log.Println(err)
		return nil, true
	}

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Println(err)
		return nil, true
	}

	taf := &taf{}

	err = json.Unmarshal(body, &taf)

	if err != nil {
		log.Println(err)
		return nil, true
	}
	return taf, false
}

func (c *context) sendMessage(message string) {
	fmt.Println(message)

	params := &openapi.CreateMessageParams{}
	params.SetTo(c.destination)
	params.SetFrom(c.sender)
	params.SetBody(message)

	resp, err := c.twilioClient.Api.CreateMessage(params)
	if err != nil {
		fmt.Println(err.Error())
	} else {
		response, _ := json.Marshal(*resp)
		fmt.Println("Response: " + string(response))
	}
}
