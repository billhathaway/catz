package main

import (
	"bytes"
	"log"
	"testing"
	"time"

	"github.com/billhathaway/strftime"
)

var (
	testController *controller
	noonKitchen    = "12:00PM"
	noonRFC3339    = "2015-01-01:12:00"
	strftimeM      = map[string]string{
		"kitchen": "%H:%M%p",
		"RFC3339": "%Y%m%dT%H:%M:%SMST",
	}
)

func init() {
	sourceTZ := "UTC"
	destTZ := "US/Pacific"
	sourceLocation, err := time.LoadLocation(sourceTZ)
	if err != nil {
		log.Fatalf("%s for sourceLocation - %s", sourceTZ, err)
	}
	destLocation, err := time.LoadLocation(destTZ)
	if err != nil {
		log.Fatalf("%s for destLocation - %s", destTZ, err)
	}
	testController = &controller{
		srcLoc:  sourceLocation,
		destLoc: destLocation,
		buf:     &bytes.Buffer{},
	}
}

// testReplace is the actual test framework
// TODO: validate the replacement values are correct versus just not returning an error
func testReplace(t *testing.T, name string, strftimeS string, origtime string) {
	var err error
	testController.format, err = strftime.New(strftimeS)
	if err != nil {
		t.Fatalf("%s failure converting %s to go time.Parse format err=%s\n", name, strftimeS, err)
		return
	}
	err = testController.strfimeToRE(strftimeS)
	if err != nil {
		t.Fatalf("%s failure converting %s to regular expression err=%s\n", name, strftimeS, err)
		return
	}
	replaced, err := testController.replaceTime(origtime)
	if err != nil {
		t.Fatalf("%s replaced returned err %s\n", name, err)
		return
	}
	t.Logf("name = %s input = %s replaced=%s\n", name, origtime, replaced)
}

// TestReplace runs all the table driven tests
func TestReplace(t *testing.T) {
	tests := []struct {
		name     string
		strftime string
		origtime string
	}{
		{"kitchen", "%H:%M%p", "12:00PM"},
		{"RFC3339", "%Y-%m-%dT%H:%M:%S", "2002-10-02T12:00:00"},
		{"syslog1", "%b %e %H:%M:%S", "Dec  5 07:52:10"},
		{"syslog2", "%b %e %H:%M:%S", "Dec 10 07:52:10"},
		{"syslog3", "%b %e %H:%M:%S", "Dec 10 13:52:10"},
		{"RFC1123Z", "%a, %d %b %Y %H:%M:%S %z", "Mon, 02 Jan 2006 15:04:05 -0700"},
	}
	for _, test := range tests {
		testReplace(t, test.name, test.strftime, test.origtime)
	}
}
