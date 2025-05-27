package httpdump

import (
	"fmt"
	"net/http"
	"net/http/httputil"

	"github.com/moul/http2curl"
)

// DumpRequest prints out an HTTP request in its wire format.
func DumpRequest(req *http.Request) string {
	var output string
	requestDump, err := httputil.DumpRequest(req, true)
	if err != nil {
		output = "++++ FAILED TO HTTP DUMP REQUEST :(\n"
	} else {
		output = fmt.Sprintf("\n++++ BEGIN DUMP HTTP REQUEST:\n%s\n++++ END DUMP HTTP REQUEST", string(requestDump))
	}
	return output
}

// DumpCURLRequest prints out an HTTP request as though it were a cURL command.
func DumpCURLRequest(req *http.Request) string {
	var output string
	curlCommand, err := http2curl.GetCurlCommand(req)
	if err != nil {
		output = output + "++++ FAILED TO DUMP CURL REQUEST :(\n"
	} else {
		output = output + fmt.Sprintf("\n++++ BEGIN DUMP CURL REQUEST:\n%s\n++++ END DUMP CURL REQUEST", curlCommand.String())
	}
	return output
}
