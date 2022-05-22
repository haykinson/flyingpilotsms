package main

import (
	"errors"
	"reflect"
	"testing"
)

func Test_getCommandFromMessage(t *testing.T) {
	tests := []struct {
		name    string
		message string
		want    *command
		wantErr bool
	}{
		{"only icao", "ICAO", &command{"ICAO", true, true}, false},
		{"3-char IAT", "IAT", &command{"IAT", true, true}, false},
		{"too short", "IC", nil, true},
		{"ICAO metar", "ICAO metar", &command{"ICAO", true, false}, false},
		{"ICAO taf", "ICAO taf", &command{"ICAO", false, true}, false},
		{"ICAO both", "ICAO metar taf", &command{"ICAO", true, true}, false},
		{"metar ICAO", "metar ICAO", &command{"ICAO", true, false}, false},
		{"taf IAT", "taf IAT", &command{"IAT", false, true}, false},
		{"multiple icao", "ICA1 ICA2 metar ICA3", &command{"ICA3", true, false}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getCommandFromMessage(tt.message)
			if (err != nil) != tt.wantErr {
				t.Errorf("getCommandFromMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getCommandFromMessage() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_onlyValidChars(t *testing.T) {
	tests := []struct {
		name string
		args string
		want bool
	}{
		{"one string", "abCd", true},
		{"with spaces", "ab cd", true},
		{"with numbers", "abc 123 def", true},
		{"with punctuation", "abc. def.", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := onlyValidChars(tt.args); got != tt.want {
				t.Errorf("onlyValidChars() = %v, want %v", got, tt.want)
			}
		})
	}
}

type testContext struct {
	t               *testing.T
	metar           *metar
	taf             *taf
	messageSent     bool
	messageContents string
}

func (c *testContext) handleMetar(string) (*metar, error) {
	if c.metar != nil {
		c.t.Logf("Returning metar: %v", c.metar)
		return c.metar, nil
	}

	c.t.Log("Returning error metar")
	return nil, errors.New("test")
}

func (c *testContext) handleTaf(string) (*taf, error) {
	if c.taf != nil {
		c.t.Logf("Returning taf: %v", c.taf)
		return c.taf, nil
	}

	c.t.Log("Returning error taf")
	return nil, errors.New("test")
}

func (c *testContext) sendMessage(message string) {
	c.messageSent = true
	c.messageContents = message
}

func Test_handleMessage(t *testing.T) {

	stdMetar := &metar{"", "metarcontents"}
	stdTaf := &taf{"", "tafcontents"}

	stdContext := &testContext{t, stdMetar, stdTaf, true, ""}

	tests := []struct {
		name           string
		context        *testContext
		message        string
		resultSent     bool
		resultContents string
	}{
		{"failed metar and taf", &testContext{t, nil, nil, false, ""}, "test", false, ""},
		{"got metar failed taf", &testContext{t, stdMetar, nil, true, ""}, "test", true, "metarcontents"},
		{"got taf failed metar", &testContext{t, nil, stdTaf, true, ""}, "test", true, "tafcontents"},
		{"got metar and taf", stdContext, "test", true, "metarcontents\n\ntafcontents"},
		{"asked for metar", stdContext, "test metar", true, "metarcontents"},
		{"asked for taf", stdContext, "test taf", true, "tafcontents"},
		{"asked for both", stdContext, "test taf metar", true, "metarcontents\n\ntafcontents"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handleMessage(tt.context, tt.message)
			if tt.resultSent != tt.context.messageSent {
				t.Errorf("Wanted sent: %v, got sent: %v", tt.resultSent, tt.context.messageSent)
				return
			}
			if tt.resultSent && tt.resultContents != tt.context.messageContents {
				t.Errorf("Wanted sent contents: %v, got sent: %v", tt.resultContents, tt.context.messageContents)
			}
		})
	}
}
