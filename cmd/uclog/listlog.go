package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	"github.com/gofrs/uuid"

	"userclouds.com/infra/namespace/color"
	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctypes/set"
	"userclouds.com/internal/servicecolors"
)

const (
	defaultColor                            = color.Default
	resubscribeTickerInterval time.Duration = 4 * time.Minute
)

var durationAndSizesExpression = regexp.MustCompile(`(?P<duration>\S+) \((?P<sz1>\d+)B -> (?P<sz2>\d+)B\)`)

// TODO This temporary code for making debugging clusters easy while we are not at huge scale
type logRecord struct {
	Service       service.Service `json:"service"`
	Tenant        string          `json:"tenant"`
	Region        string          `json:"r"`
	Host          string          `json:"h"`
	Content       string          `json:"content"`
	Code          uclog.EventCode `json:"type"`
	Payload       string          `json:"payload"`
	Timestamp     int             `json:"timestamp"`
	Applicationid string          `json:"applicationId"`
}

type perfRecord struct {
	totalTime              int64
	dbTime                 int64
	cacheTime              int64
	queueTime              int64
	dbCalls                int
	cacheHits              int
	cacheMisses            int
	compressedOutputSize   int
	uncompressedOutputSize int
}

type listLogConfig struct {
	writeLocalStream bool
	filename         string
	fileHandle       *os.File
	fileWriter       *bufio.Writer
}

var settings listLogConfig
var writeMutex sync.Mutex
var skipMapURL = make(map[uuid.UUID]bool)
var skipMapTenant = make(map[uuid.UUID]bool)
var records = make([]logRecord, 0, 10000)
var filterOutServices = map[service.Service]bool{}
var filteredRequests = map[string]int{}
var slowCallLimit int64
var callSummary = map[string]int{}
var failedCalls = []logRecord{}
var slowCalls = []logRecord{}
var tenantCalls = map[string]int{}
var hostCalls = map[string]int{}
var regionCalls = map[string]int{}
var loginCalls = map[string]int{}
var verbose bool
var onlyIncludeTenant uuid.UUID
var onlyIncludeRegion string
var onlyIncludeService service.Service
var perfMonValues = map[string][]perfRecord{}
var perfMonitorPrefix string
var outputPrefix string
var httpIgnoreCodesMap = map[int]bool{}
var httpIgnoreCodesCount = map[int]int{}
var tenantURLs = map[string]string{}
var tenantIDs = map[uuid.UUID]uuid.UUID{}
var outputLogDataToScreen = false

func generatePrefix(lR *logRecord, requestID uuid.UUID, interactive bool) string {
	serviceLabel := string(lR.Service)
	if interactive {
		serviceLabel = fmt.Sprintf("%s%s%s%s%s", color.ANSIEscapeColor, servicecolors.Colors[lR.Service], lR.Service, color.ANSIEscapeColor, defaultColor)
	}

	tm := time.Unix(int64(lR.Timestamp/1000), 0)

	prefix := ""

	for _, pt := range outputPrefix {
		switch pt {
		case 'h':
			prefix = fmt.Sprintf("%s[%s]", prefix, lR.Host)
		case 'r':
			prefix = fmt.Sprintf("%s[%s]", prefix, lR.Region)
		case 't':
			if tenantID, ok := tenantIDs[requestID]; ok {
				prefix = fmt.Sprintf("%s[%v]", prefix, tenantID)
			} else {
				prefix = fmt.Sprintf("%s[%s]", prefix, lR.Tenant)
			}
		case 's':
			prefix = fmt.Sprintf("%s%-8s ", prefix, serviceLabel)

		}
	}

	return fmt.Sprintf("%s[%s]", prefix, tm.Format("Jan _2 15:04:05"))
}

func outputLogData(ctx context.Context, interactive bool) int {
	// If there are no records to display, return 0
	if len(records) == 0 {
		return 0
	}

	// Sort the records by time to make them easier to read
	sort.SliceStable(records, func(i, j int) bool {
		return records[i].Timestamp < records[j].Timestamp
	})

	var displayedRecords = 0
	for _, lR := range records {
		requestID, err := uuid.FromString(lR.Content[:strings.Index(lR.Content, ":")])
		if err != nil {
			uclog.Errorf(ctx, "failed to parse uuid: %v", lR.Content)
		}

		if !skipMapURL[requestID] && !skipMapTenant[requestID] {

			prefix := generatePrefix(&lR, requestID, interactive)
			if outputLogDataToScreen {
				uclog.Infof(ctx, "%s: %s", prefix, lR.Content) // goes to both screen and file file transports
			} else {
				uclog.Verbosef(ctx, "%s: %s", prefix, lR.Content) // only goes to the file transport
			}
			displayedRecords++
		}
	}

	records = []logRecord{}

	return displayedRecords
}

func outputSummary(ctx context.Context, interactive bool, summary bool, callsummary bool, displayedRecords *int, runtime int) {
	var summaryString = fmt.Sprintf("Runtime %d secs. Filtered out - ", runtime)
	// Only display filtered out call if there are not too many of them
	if len(filteredRequests) < 10 {
		for s, fc := range filteredRequests {
			summaryString = fmt.Sprintf("%s %s: %d", summaryString, s, fc)
		}
	} else {
		summaryString = fmt.Sprintf("%s too many to list", summaryString)
	}

	summaryString = fmt.Sprintf("%s Displayed: %d Failed Http Calls: %d", summaryString, *displayedRecords, len(failedCalls))
	uclog.Infof(ctx, "%s", summaryString)

	if callsummary {
		uclog.Infof(ctx, "Call summary:")
		for s, fc := range callSummary {
			uclog.Infof(ctx, "%s: %d", s, fc)
		}
	}

	if summary {
		uclog.Infof(ctx, "Tenant call counts:")
		for t, tc := range tenantCalls {
			tenantURL, ok := tenantURLs[t]
			if !ok {
				tenantURL = "unknown"
			}
			uclog.Infof(ctx, "%s: [%s] %d", t, tenantURL, tc)
		}

		uclog.Infof(ctx, "Host log lines:")
		for t, tc := range hostCalls {
			uclog.Infof(ctx, "%s: %d", t, tc)
		}

		uclog.Infof(ctx, "Region log lines:")
		for t, tc := range regionCalls {
			uclog.Infof(ctx, "%s: %d", t, tc)
		}
		uclog.Infof(ctx, "Logins:")
		for l, lc := range loginCalls {
			uclog.Infof(ctx, "%s: %d", l, lc)
		}
		if len(loginCalls) == 0 {
			uclog.Infof(ctx, "No logins during the time period")
		}

		uclog.Infof(ctx, "Failed calls:")
		for code, count := range httpIgnoreCodesCount {
			uclog.Infof(ctx, "Ignored http %d: %d", code, count)
		}
		for _, lR := range failedCalls {
			requestID, err := uuid.FromString(lR.Content[:strings.Index(lR.Content, ":")])
			if err != nil {
				uclog.Errorf(ctx, "failed to parse uuid: %v", lR.Content)
			}

			if !skipMapURL[requestID] && !skipMapTenant[requestID] {
				prefix := generatePrefix(&lR, requestID, interactive)
				uclog.Infof(ctx, "%s: %s", prefix, lR.Content)
			}
		}
		if len(failedCalls) == 0 {
			uclog.Infof(ctx, "No failed calls during the time period")
		}

		if slowCallLimit > 0 {
			uclog.Infof(ctx, "Slow (over %dms) calls - %d :", slowCallLimit, len(slowCalls))
			for _, lR := range slowCalls {
				requestID, err := uuid.FromString(lR.Content[:strings.Index(lR.Content, ":")])
				if err != nil {
					uclog.Errorf(ctx, "failed to parse uuid: %v", lR.Content)
					continue
				}

				if !skipMapURL[requestID] && !skipMapTenant[requestID] {
					prefix := generatePrefix(&lR, requestID, interactive)
					uclog.Infof(ctx, "%s: %s", prefix, lR.Content)
				}
			}
			if len(slowCalls) == 0 {
				uclog.Infof(ctx, "No slow calls during the time period")
			}
		}
	}

	// Output performance summary data
	if perfMonitorPrefix != "" {
		for u, perfVals := range perfMonValues {
			numOpsTotal := len(perfVals)
			// Sort the array
			sort.Slice(perfVals, func(i, j int) bool {
				return perfVals[i].totalTime < perfVals[j].totalTime
			})
			// Get total time spent
			var totalPR perfRecord
			for _, v := range perfVals {
				totalPR.totalTime = totalPR.totalTime + v.totalTime
				totalPR.dbTime = totalPR.dbTime + v.dbTime
				totalPR.cacheTime = totalPR.cacheTime + v.cacheTime
				totalPR.queueTime = totalPR.queueTime + v.queueTime
				totalPR.dbCalls = totalPR.dbCalls + v.dbCalls
				totalPR.cacheHits = totalPR.cacheHits + v.cacheHits
				totalPR.cacheMisses = totalPR.cacheMisses + v.cacheMisses
			}

			p99index := int(float32(numOpsTotal) * 0.99)
			if p99index >= len(perfVals) {
				p99index = len(perfVals) - 1
			}
			p9999index := int(float32(numOpsTotal) * 0.9999)
			if p9999index >= len(perfVals) {
				p9999index = len(perfVals) - 1
			}
			p9999 := float64(perfVals[p9999index].totalTime) / 1000
			p99 := float64(perfVals[p99index].totalTime) / 1000
			p90 := float64(perfVals[int(float32(numOpsTotal)*0.9)].totalTime) / 1000
			p50 := float64(perfVals[int(float32(numOpsTotal)*0.5)].totalTime) / 1000
			p10 := float64(perfVals[int(float32(numOpsTotal)*0.1)].totalTime) / 1000
			uclog.Infof(ctx, "Perf Summary: %4d calls to %-40s | op p10 %.2f ms | op p50 %.2f ms | op p90 %.2f ms | op p99 %.2f ms | op p99.99 %.2f ms | ave t %.2f ms, db t %.2f ms, cache t %.2f ms, wait t %.2f ms, db calls %.2f, cache hits %.2f, cache misses %.2f",
				numOpsTotal, u, p10, p50, p90, p99, p9999,
				float64(totalPR.totalTime)/float64(numOpsTotal)/1000,
				float64(totalPR.dbTime)/float64(numOpsTotal)/1000,
				float64(totalPR.cacheTime)/float64(numOpsTotal)/1000,
				float64(totalPR.queueTime)/float64(numOpsTotal)/1000,
				float64(totalPR.dbCalls)/float64(numOpsTotal),
				float64(totalPR.cacheHits)/float64(numOpsTotal),
				float64(totalPR.cacheMisses)/float64(numOpsTotal))
		}

		if len(perfMonValues) == 0 {
			uclog.Infof(ctx, "Perf Summary: no calls to %s for perf summary", perfMonitorPrefix)
		}
	}
	if settings.writeLocalStream {
		uclog.Infof(ctx, "Wrote raw logs to %s", settings.filename)
	}
}

func unmarshalAndProcess(ctx context.Context, t *time.Time, r []byte, record *uclog.LogRecordArray) error {
	// If request write the bytes read from kinesis stream to a file
	if settings.writeLocalStream && r != nil {
		settings.fileWriter.Write(r)
	}

	var rA uclog.LogRecordArray
	if record == nil {
		if err := json.Unmarshal(r, &rA); err != nil {
			uclog.Errorf(ctx, "Error %v Didn't parse %s : %s ", err, t.String(), string(r))
			return nil
		}
	} else {
		rA = *record
	}

	for i := range rA.Records {
		if err := processRecord(ctx, logRecord{Service: rA.Service, Tenant: rA.TenantID.String(), Host: rA.Host, Region: string(rA.Region),
			Content: rA.Records[i].Message, Payload: rA.Records[i].Payload, Code: rA.Records[i].Code, Timestamp: rA.Records[i].Timestamp}); err != nil {
			return ucerr.Wrap(err)
		}
	}
	return nil
}

var (
	skippedRequests = set.NewStringSet("GET /healthcheck", "GET /", "GET /SiteLoader", "GET /readiness")
)

// Process a single record from the stream
// TODO (sgarrity 8/23): oof...this should be unified and tested with log emission?
func processRecord(ctx context.Context, lR logRecord) error {

	//  We will error no non-lower case name here but for now this allows the code to work against prod
	lR.Service = service.Service(strings.ToLower(string(lR.Service)))

	// Check if this service is not being tracked and stop processing in that case
	if v, ok := filterOutServices[lR.Service]; ok && v || onlyIncludeService != "" && onlyIncludeService != lR.Service {
		filteredRequests[string(lR.Service)] = filteredRequests[string(lR.Service)] + 1
		return nil
	}
	// Check if this region is not being tracked and stop processing in that case
	lR.Region = strings.ToLower(lR.Region)
	if onlyIncludeRegion != "" && lR.Region != onlyIncludeRegion {
		filteredRequests[lR.Region] = filteredRequests[lR.Region] + 1
		return nil
	}

	// Filter out events without messages TODO add some convenient functions here
	if lR.Content == "" {
		return nil
	}
	requestIDIndex := strings.Index(lR.Content, ":")
	if requestIDIndex == -1 {
		uclog.Errorf(ctx, "Can't parse request id from '%s'", lR.Content)
		return nil
	}
	requestID, err := uuid.FromString(lR.Content[:requestIDIndex])
	if err != nil {
		uclog.Errorf(ctx, "Didn't parse request id from '%s'", lR.Content)
		return nil
	}

	tenantID, err := uuid.FromString(lR.Tenant)
	if err != nil {
		uclog.Errorf(ctx, "Didn't parse tenant id from %s in '%s'", lR.Tenant, lR.Content)
		return nil
	}

	// If we are interested only in one tenant - filter calls from other tenants out
	if onlyIncludeTenant != uuid.Nil {
		if onlyIncludeTenant != tenantID {
			skipMapTenant[requestID] = true
		} else {
			skipMapTenant[requestID] = false
		}
	}

	if !tenantID.IsNil() {
		if _, ok := tenantIDs[requestID]; !ok {
			tenantCalls[lR.Tenant]++
		}
		tenantIDs[requestID] = tenantID
	}

	hostCalls[lR.Host]++
	regionCalls[lR.Region]++

	message := lR.Content[strings.Index(lR.Content, ":"):]
	// Look for HttpRequest events and use them to filter http request by URL being called
	if lR.Code == 99 || lR.Code == -1 {
		// 0fc47476-9ece-4b1b-a93a-0ec2b1841849: HTTP request | [] | GET /healthcheck
		// or
		// d71e6938-2407-4362-9cc3-05ff7f7a2f49: HTTP request | [] | GET /authz/objects/9d30c731-1ce4-4a48-9ab2-d677330e5b93
		// or
		// b5832b12-655b-4605-a6a5-d56bd6c5ac36: HTTP request | [] | GET /objects/b9e4eca8-88c5-4be6-b4b8-d7c60f0f19cb | tenant 9eb68c73-4bd3-43c1-840f-547e4e211c7f
		// or (older, before https://github.com/userclouds/userclouds/commit/46db1421ae6d2cee37624198b923744682a70f42#diff-5698d7c69b3e1c107a692db252bed1c714ff069af252aa78f81db6b36be6534eR159)
		// b5832b12-655b-4605-a6a5-d56bd6c5ac36: HTTP request | GET /objects/b9e4eca8-88c5-4be6-b4b8-d7c60f0f19cb | tenant 9eb68c73-4bd3-43c1-840f-547e4e211c7f
		// or
		// // 0fc47476-9ece-4b1b-a93a-0ec2b1841849: HTTP request | GET /healthcheck
		parts := strings.Split(message, "|")
		lastPart := strings.TrimSpace(parts[len(parts)-1])
		if hasTenant := strings.HasPrefix(lastPart, "tenant"); !hasTenant {
			request := lastPart // 1st & 2nd examples above
			if skippedRequests.Contains(request) {
				skipMapURL[requestID] = true
				filteredRequests[request] = filteredRequests[request] + 1
			} else { // Skip the double logging of lines with | tenantid in them for the call summary
				callSummary[request] = callSummary[request] + 1
			}
		}
	}

	// Look for httpResponseEvents and record timing information if requested
	if lR.Code >= 100 && lR.Code <= 599 {
		// Record failed requests
		if lR.Code >= 400 && lR.Code <= 599 {
			if !httpIgnoreCodesMap[int(lR.Code)] {
				failedCalls = append(failedCalls, lR)
			} else {
				httpIgnoreCodesCount[int(lR.Code)]++
			}
		}

		//:HTTP     200 | 202.653ms (1234B) | DB 2 10.104ms | Cache: c:154 h:152 m:1 48.253ms | tenant.userclouds.com | 5c50376d-db44-4569-9a19-a93f26398e2f | POST /tokenizer/tokens/actions/resolve
		// or
		//:HTTP     200 | 202.653ms (1234B -> 100B) | DB 2 10.104ms | Cache: c:154 h:152 m:1 48.253ms | tenant.userclouds.com | 5c50376d-db44-4569-9a19-a93f26398e2f | POST /tokenizer/tokens/actions/resolve
		pR := perfRecord{}
		parts := strings.Split(message, "|")
		durationAndSize := strings.TrimSpace(parts[1])
		var duration string
		if strings.Contains(durationAndSize, " -> ") {
			// New format
			matches := durationAndSizesExpression.FindStringSubmatch(durationAndSize)
			if len(matches) != 4 { // 0 is the whole match, 1-3 are the marched groups
				uclog.Errorf(ctx, "failed to parse duration and sizes from '%s' (line: %s)", durationAndSize, lR.Content)
			}
			duration = matches[1]
			pR.uncompressedOutputSize, err = strconv.Atoi(matches[2])
			if err != nil {
				uclog.Errorf(ctx, "Didn't parse uncompressed output size from %s in %s", durationAndSize, lR.Content)
				return nil
			}
			pR.compressedOutputSize, err = strconv.Atoi(matches[3])
			if err != nil {
				uclog.Errorf(ctx, "Didn't parse compressed output size from %s in %s", durationAndSize, lR.Content)
				return nil
			}
		}

		dbData := strings.TrimSpace(parts[2])
		dbCalls := strings.Split(dbData, " ")[1]
		pR.dbCalls, err = strconv.Atoi(dbCalls)
		if err != nil {
			uclog.Errorf(ctx, "Didn't parse DB call count  from %s in %s", dbCalls, lR.Content)
			return nil
		}
		dbDuration := strings.Split(dbData, " ")[2]
		dbTime, err := time.ParseDuration(dbDuration)
		if err != nil {
			uclog.Errorf(ctx, "Didn't parse DB duration count  from %s in %s", dbCalls, lR.Content)
			return nil
		}
		pR.dbTime = dbTime.Microseconds()

		offset := 3
		if strings.HasPrefix(parts[offset], " Cache: ") {
			cacheParts := strings.Split(strings.TrimSpace(parts[offset]), " ")
			cacheHits := cacheParts[2]
			pR.cacheHits, err = strconv.Atoi(strings.TrimPrefix(cacheHits, "h:"))
			if err != nil {
				uclog.Errorf(ctx, "Didn't parse cache hit count from %s in %s, with %v", cacheHits, lR.Content, strings.Split(parts[offset], " "))
				for i, s := range strings.Split(parts[offset], " ") {
					uclog.Errorf(ctx, "%d %s", i, s)
				}
				return nil
			}
			cacheMisses := cacheParts[3]
			pR.cacheMisses, err = strconv.Atoi(strings.TrimPrefix(cacheMisses, "m:"))
			if err != nil {
				uclog.Errorf(ctx, "Didn't parse cache miss count from %s in %s with %s", cacheMisses, lR.Content, cacheMisses)
				return nil
			}
			cacheDuration := cacheParts[4]
			cacheTime, err := time.ParseDuration(cacheDuration)
			if err != nil {
				uclog.Errorf(ctx, "Didn't parse cache duration from %s in %s", cacheDuration, lR.Content)
				return nil
			}
			pR.cacheTime = cacheTime.Microseconds()
			offset++
		}

		tenantURL := strings.TrimSpace(parts[offset])

		if strings.HasPrefix(parts[offset+2], " W: ") {
			waitTimeStr := strings.TrimSpace(strings.TrimPrefix(parts[offset+2], " W: "))

			queueWait, err := time.ParseDuration(waitTimeStr)
			if err != nil {
				uclog.Errorf(ctx, "Didn't parse wait time duration from %s in %s", waitTimeStr, lR.Content)
				return nil
			}
			pR.queueTime = queueWait.Microseconds()
			offset++
		}
		url := strings.TrimSpace(parts[offset+2])

		callDur, err := time.ParseDuration(duration)
		if err != nil {
			uclog.Errorf(ctx, "Didn't parse duration from %s in %s", durationAndSize, lR.Content)
			return nil
		}
		pR.totalTime = callDur.Microseconds()
		if pR.totalTime > slowCallLimit*1000 {
			slowCalls = append(slowCalls, lR)
		}

		tenantURLs[lR.Tenant] = tenantURL
		// If the prefix doesn't include an HTTP verb like POST or GET do the match on the url only
		// which allows a url prefix to match a subtree like /tokenizer
		// TODO add ability to template ids in the url to combine operations on individual items

		if strings.Contains(url, strings.TrimSuffix(strings.TrimSuffix(perfMonitorPrefix, "?"), "*")) && !skipMapTenant[requestID] {
			urlIndex := url
			if strings.HasSuffix(perfMonitorPrefix, "?") {
				urlIndex = strings.Split(urlIndex, "?")[0]
			}
			if strings.HasSuffix(perfMonitorPrefix, "*") {
				urlIndex = perfMonitorPrefix
			}

			if _, ok := perfMonValues[urlIndex]; !ok {
				perfMonValues[urlIndex] = make([]perfRecord, 0)
			}
			perfMonValues[urlIndex] = append(perfMonValues[urlIndex], pR)
		}
	}

	// Trim verbose messages if requested
	if !verbose && strings.HasPrefix(message, ": [V]") {
		return nil
	}
	// Detect [I] Plex [Aug 10 15:35:48]: 3b5f6c43-3800-43c4-9d1d-138637ff9b1a: [I] login session for vlad-11-11@foo.net
	// TODO this should be a security event with id as payload
	if lR.Service == "plex" && strings.HasPrefix(message, ": [I] login session for ") {
		loginid := strings.TrimSpace(strings.TrimPrefix(message, ": [I] login session for "))
		loginCalls[loginid] = loginCalls[loginid] + 1
	}
	if v, ok := skipMapURL[requestID]; !ok || !v {
		records = append(records, lR)
	}
	return nil
}

func processFile(ctx context.Context, inputFileName string) error {
	fileHandle, err := os.Open(inputFileName)
	if err != nil {
		uclog.Fatalf(ctx, "Couldn't open temporary file %s for raw logs - %v", settings.filename, err)
	}
	defer fileHandle.Close()
	decoder := json.NewDecoder(fileHandle)
	for {
		var data uclog.LogRecordArray
		err = decoder.Decode(&data)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			uclog.Errorf(ctx, "Failed to decode input file %s - %v", inputFileName, err)
			return ucerr.Wrap(err)
		}
		err = unmarshalAndProcess(ctx, &time.Time{}, nil, &data)
		if err != nil {
			uclog.Errorf(ctx, "Failed to process %v", data)
		}
	}
	return nil
}

// ListRegionLogsForService lists all the log messages in the kinesis stream for [tenant, service, region]
// TODO: this function needs to take a sane number of arguments -> filter struct?
func ListRegionLogsForService(ctx context.Context,
	sc *streamsClient,
	tenantID uuid.UUID,
	region string,
	svc service.Service,
	timeWindow string,
	streamName string,
	interactive bool,
	serviceFilter string,
	verboseArg bool,
	callsummary bool,
	summary bool,
	slowCallLimitArg int,
	live bool,
	perfurl string,
	oprefix string,
	httpIgnoreCodes string,
	inputFileName string,
	writeRawLogs bool,
	outputLogDataToScreenArg bool,
) error {

	// Start time
	funcStartTime := time.Now().UTC()
	// Interpret time window
	timeWindowSplit := strings.Split(timeWindow, ",")
	if len(timeWindowSplit) != 1 && len(timeWindowSplit) != 2 {
		uclog.Fatalf(ctx, "Invalid time window - %s", timeWindow)
	}
	timeWindowStart, err := strconv.Atoi(timeWindowSplit[0])
	if err != nil {
		uclog.Fatalf(ctx, "Invalid time window start- %s", timeWindow)
	}
	timeWindowEnd := 0
	if len(timeWindowSplit) == 2 {
		timeWindowEnd, err = strconv.Atoi(timeWindowSplit[1])
		if err != nil {
			uclog.Fatalf(ctx, "Invalid time window end - %s", timeWindow)
		}
	}
	// Initialize filters
	sF := strings.SplitSeq(serviceFilter, ",")
	for f := range sF {
		if f != "" {
			fs := service.Service(f)
			if !service.IsValid(fs) {
				uclog.Fatalf(ctx, "Invalid service in filter - %s", f)
			}
			filterOutServices[fs] = true
		}
	}
	onlyIncludeService = svc
	if onlyIncludeService != "" {
		uclog.Infof(ctx, "Filtering logs to only calls to service - %s", onlyIncludeService)
	}
	if !tenantID.IsNil() {
		onlyIncludeTenant = tenantID
		uclog.Infof(ctx, "Filtering logs to only calls to tenant - %v", onlyIncludeTenant)
	}
	onlyIncludeRegion = strings.ToLower(region)
	if onlyIncludeRegion != "" {
		uclog.Infof(ctx, "Filtering logs to only calls to region - %s", onlyIncludeRegion)
	}

	if httpIgnoreCodes != "" {
		codesSplit := strings.SplitSeq(httpIgnoreCodes, ",")
		for cs := range codesSplit {
			cn, err := strconv.Atoi(cs)
			if err != nil || cn < 400 || cn > 600 {
				uclog.Fatalf(ctx, "Invalid http ignore code(s) in - %s. Expect code >= 400 and < 600", httpIgnoreCodes)
			}
			httpIgnoreCodesMap[cn] = true
			httpIgnoreCodesCount[cn] = 0
		}
		uclog.Infof(ctx, "Not treating http codes %s as errrors", httpIgnoreCodes)
	}

	verbose = verboseArg
	outputLogDataToScreen = outputLogDataToScreenArg
	perfMonitorPrefix = perfurl
	outputPrefix = oprefix
	slowCallLimit = int64(slowCallLimitArg)

	if writeRawLogs {
		initRawFile(ctx)

		defer settings.fileWriter.Flush()
		defer settings.fileHandle.Sync()
		defer settings.fileHandle.Close()
	}
	// Figure out the start time from which to read logs and process data up to tip
	startTime := time.Now().UTC().Add(time.Duration(-int64(timeWindowStart) * int64(time.Minute)))
	endTime := time.Now().UTC().Add(time.Duration(-int64(timeWindowEnd) * int64(time.Minute)))
	// Make sure start and end time are not reversed
	if startTime.After(endTime) {
		uclog.Fatalf(ctx, "Start time %v is after end time %v", startTime, endTime)
	}
	uclog.Infof(ctx, "Getting logs for time window [%v, %v]", startTime, endTime)
	var shardPosition = map[string]string{}
	var streamDesc *types.StreamDescription

	if inputFileName == "" {
		shardPosition, streamDesc, err = sc.getShardPositions(ctx, startTime, endTime, timeWindowEnd)
		if err != nil {
			return ucerr.Wrap(err)
		}
	} else {
		err = processFile(ctx, inputFileName)
		if err != nil {
			uclog.Fatalf(ctx, "Failed to process input file %s due to error %v", inputFileName, err)
		}
	}

	displayedRecords := outputLogData(ctx, interactive)

	if live {
		handler := func(ctx context.Context, t *time.Time, r []byte, record *uclog.LogRecordArray) error {
			if err := unmarshalAndProcess(ctx, t, r, record); err != nil {
				return ucerr.Wrap(err)
			}
			displayedRecords += outputLogData(ctx, interactive)
			return nil
		}
		doneChannels, err := sc.startLive(ctx, streamDesc, startTime, shardPosition, handler)
		if err != nil {
			return ucerr.Wrap(err)
		}
		sigchan := make(chan os.Signal, 100)
		signal.Notify(sigchan, syscall.SIGTERM, // "the normal way to politely ask a program to terminate"
			syscall.SIGINT,  // Ctrl+C
			syscall.SIGQUIT, // Ctrl-\
			syscall.SIGHUP)  // "terminal is disconnected")
		<-sigchan

		// Signal all the worker threads to stop
		uclog.Infof(ctx, "Signaling %d thread(s) to exit", len(doneChannels))
		for _, done := range doneChannels {
			done <- true
		}
	}

	runtime := int(time.Now().UTC().Sub(funcStartTime).Seconds())
	outputSummary(ctx, interactive, summary, callsummary, &displayedRecords, runtime)
	return ucerr.Wrap(err)
}

func initRawFile(ctx context.Context) {
	settings.writeLocalStream = true

	f, err := os.CreateTemp("/tmp/", "rawlogs"+".*")
	if err != nil {
		uclog.Fatalf(ctx, "Couldn't create a temporary file for raw logs - %v", err)
	}
	settings.filename = f.Name()
	f.Close()

	settings.fileHandle, err = os.OpenFile(settings.filename, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		uclog.Fatalf(ctx, "Couldn't open temporary file %s for raw logs - %v", settings.filename, err)
	}
	settings.fileWriter = bufio.NewWriter(settings.fileHandle)
}
