package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/uclog"
)

/*

This is a temporary convinience script. To use it paste the lines from running the service that describe missing events for handlers into
a file. Then determine the full service name that you want to appear in LogEventTypeInfo and pass that as second argument. So if you pass in
"Plex" then for handler "loginHandler-fm", the script will generate "Plex.loginHandler-fm.Count" and "Plex.loginHandler-fm.Duration". Then
decide on the short service name for event codes. If you select "Plex" the event names for same handler will be generates
as "EventPlexLoginHandler" and  "EventPlexLoginHandlerDuration". Get the first avaliable event code like 1510 by looking at events.go.

Examples:

eventsgen Plex plex 1510 missinghandlers
eventsgen CNSL console 1510 missinghandlers
eventsgen CNSL console 1510 missinghandlers
eventsgen Authz authz 1510 missinghandlers

The output can be pasted directly into events.go and you just need to fill in the event name instead of TBD and the URL

*/

func main() {
	ctx := context.Background()

	options := []logtransports.ToolLogOption{}

	options = append(options, logtransports.Prefix(0))

	logtransports.InitLoggerAndTransportsForTools(ctx, uclog.LogLevelInfo, uclog.LogLevelVerbose, "eventsgen", options...)
	defer logtransports.Close()

	if len(os.Args) < 4 {
		uclog.Fatalf(ctx, "Usage: eventsgen [serviceShortName] [serviceFullName] [starting event code] [filename]")
	}

	serviceName := os.Args[1]
	serviceNameFull := os.Args[2]

	startCode, err := strconv.Atoi(os.Args[3])
	if err != nil || startCode < 0 || startCode%10 != 0 {
		uclog.Fatalf(ctx, "Failed to parse starting event code %s or not divisible by 10 as expected: %v", os.Args[3], err)
	}

	filename := os.Args[4]
	f, err := os.Open(filename)
	if err != nil {
		uclog.Fatalf(ctx, "failed to open file with missing handler names %v: %v", filename, err)
	}

	uclog.Debugf(ctx, "Opening file: %s Starting Event Codes at %d for service %s ", filename, startCode, serviceNameFull)

	defer f.Close()

	var events []string

	// read the file line by line using scanner
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		input := scanner.Text()
		line := strings.Split(input, " ")
		// Skip lines that are not about the handlers
		if len(line) < 7 || !strings.Contains(input, "event for handler") {
			continue
		}
		nameHandler := line[len(line)-1]
		caser := cases.Title(language.AmericanEnglish)
		nameHandlerTrimmed := strings.Join(strings.Split(caser.String(strings.TrimSuffix(nameHandler, "-fm")), "."), "")
		nameEvent := "Event" + serviceName + nameHandlerTrimmed

		// Skip duplicates
		skip := false
		for _, s := range events {
			if s == nameEvent {
				skip = true
			}
		}
		if skip {
			continue
		}

		uclog.Infof(ctx, "\"%s.Count\": {Name: \"TBD\", Code: %s, Service: \"%s\", URL: \"\", Type: \"Call\"},", nameHandler,
			nameEvent, serviceNameFull)

		nameEventDuration := "Event" + serviceName + strings.TrimSuffix(nameHandlerTrimmed, ".Func1") + "Duration"
		uclog.Infof(ctx, "\"%s.Duration\": {Name: \"TBD\", Code: %s, Service: \"%s\", URL: \"\", Type: \"Duration\"},\n", nameHandler,
			nameEventDuration, serviceNameFull)
		events = append(events, nameEvent, nameEventDuration)
	}

	// Check if there was a file error
	if err := scanner.Err(); err != nil {
		uclog.Fatalf(ctx, "Error reading file %s - %v", filename, err)
	}

	eventCode := startCode
	for _, s := range events {
		fmt.Printf("%s          int = %d\n", s, eventCode)
		if eventCode%10 == 0 {
			eventCode++
		} else {
			eventCode = eventCode + 9
		}
	}
}
