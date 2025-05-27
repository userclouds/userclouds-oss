package security

import (
	"context"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-http-utils/headers"

	"userclouds.com/infra/middleware"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uclog/responsewriter"
)

type contextKey int

const ctxSecurityStatus contextKey = 1 // key for security status

// GC interval
const (
	// This allows for 100 more failing calls than successful calls per IP each 2 seconds if the caller back off first 401 response
	// After that the penalty grows for each attempted call during the blocked period
	penaltyPerBlockedCall time.Duration = 1 * time.Second
	// This allows for 10 more failing calls than successful calls per user each 61 seconds if the caller back off first 401 response
	// After that the penalty grows for each attempted call during the blocked period
	penaltyPerBlockedUser    time.Duration = 60 * time.Second
	expirationPerBlockedCall time.Duration = 60 * time.Second
	expirationPerBlockedUser time.Duration = 120 * time.Second
	expireInterval           time.Duration = 60 * time.Second

	callFailureLimitPerIP   = 100
	callFailureLimitPerUser = 10
)

// ReqValidator interface for object able to validate request
// TODO: is username the right identifier here? We have User ID (a UUIDv4 in our IDP) but other IDPs
// use OIDC Subject. For passwordless login there's emails and eventually SMS numbers. If we just need a unique
// key, then this is probably fine.
type ReqValidator interface {
	ValidateRequest(r *http.Request) (Status, error)
	FinishedRequest(status Status, r *http.Request, success bool)
	IsCallBlocked(ctx context.Context, username string) bool
}

// Code type for int codes for different security states
type Code int

// Security event names TODO - this is clumsy will be fixed in next update of security code
const eventInternalValidationError = "Security.InternalError"
const eventRequestBlockedByIP = "Security.IPBlocked"
const eventRequestBlockedByUserName = "Security.UserBlocked"
const eventFailureByIP = "Security.IPFail"
const eventFailureByUserName = "Security.UserFail"
const eventRequestBlockedByIPInHostHeader = "Security.IPBlockedInHostHeader"

// Status maintains the current client side status of security check for a request
type Status struct {
	Code     string
	Message  string
	Username string
	IPs      []string
}

type callHistory struct {
	count        int
	lastActive   time.Time
	blockedUntil time.Time
}

// securityChecker validates that response should be processed
type securityChecker struct {
	done         chan bool
	ipMutex      sync.Mutex
	userMutex    sync.Mutex
	expireTicker time.Ticker
	ipHistory    map[string]*callHistory
	userHistory  map[string]*callHistory
}

// GetSecurityStatus returns current security status for this request if one was set
func GetSecurityStatus(ctx context.Context) *Status {
	val := ctx.Value(ctxSecurityStatus)
	id, ok := val.(*Status)
	if !ok {
		return nil
	}
	return id
}

// SetSecurityStatus returns a Tenant if one is set for this context and nil otherwise
func SetSecurityStatus(ctx context.Context, s *Status) context.Context {
	return context.WithValue(ctx, ctxSecurityStatus, s)
}

// NewSecurityChecker initializes the global security checker across all requests
func NewSecurityChecker() ReqValidator {
	var checker securityChecker
	// Call history maps. Server hints are inserted into these maps as well as local calls
	checker.ipHistory = make(map[string]*callHistory)
	checker.userHistory = make(map[string]*callHistory)

	// Create mutexes to protect read and write access to the maps since it is not thread safe
	checker.ipMutex = sync.Mutex{}
	checker.userMutex = sync.Mutex{}

	// Initialize a timer to expire records from the histories
	checker.expireTicker = *time.NewTicker(expireInterval)
	checker.done = make(chan bool)
	go func() {
		for {
			select {
			case <-checker.done:
				return
			case <-checker.expireTicker.C:
				checker.expireIPHistory()
				checker.expireUserHistory()
			}
		}
	}()

	return &checker
}

// Middleware is a wrapper that runs security checks before the request is processed
func Middleware(reqChecker ReqValidator) middleware.Middleware {
	return middleware.Func((func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// If request pass security checks continue to process request otherwise about
			s, err := reqChecker.ValidateRequest(r)
			ctx = SetSecurityStatus(ctx, &s)
			if err != nil {
				// TODO: if security checks fail, we currently return 204 so that we don't
				// screw up our ALB health metrics (we get alerts based on 4xx / 5xx rates)
				// with real volume we should fix this to 401 probably?
				w.WriteHeader(http.StatusNoContent)
				uclog.IncrementEvent(ctx, s.Code)
				return
			}
			lrw := responsewriter.NewStatusResponseWriter(w)
			next.ServeHTTP(lrw, r.WithContext(ctx))
			reqChecker.FinishedRequest(s, r, (lrw.StatusCode < 400))
		})
	}))
}

func (s *securityChecker) ValidateRequest(r *http.Request) (Status, error) {
	ctx := r.Context()

	var status = Status{}
	ips, err := getIPs(r)
	// If we failed to figure out the IP address for the call - fail the request
	if err != nil {
		status.Code = eventInternalValidationError
		return status, ucerr.New("Failed to get IP addresses")
	}

	// check if IP is blocked, if so block the request and log the local block event to the server
	for _, ip := range ips {
		if s.checkForBlockedIP(ip) {
			status.Code = eventRequestBlockedByIP
			uclog.IncrementEventWithPayload(ctx, eventRequestBlockedByIP, ip)
			return status, ucerr.New("Rate limits exceeded")
		}
	}
	status.IPs = ips

	// TODO check if the headers are suspect against the cache, if so block the request and log the local block event to the server
	// TODO should drop requests if "user-agent" is blank - Google 403s. should we drop requests that case through port 80
	// on the load balancer "x-forwarded-port" != 443
	if userAgent := r.UserAgent(); userAgent == "" {
		uclog.Debugf(ctx, "Headers: Got request with blank user-agent")
	}
	if fPort := r.Header.Get("x-forwarded-port"); fPort != "443" && !universe.Current().IsDev() {
		uclog.Debugf(ctx, "Headers: Got request from port %s", fPort)
	}

	// if the host header is an IP address (not a hostname), block the request
	// since our system always requires a per-tenant hostname in the host header
	// note that r.Host can be host:port so we handle that as needed
	host := r.Host
	parts := strings.Split(r.Host, ":")
	if len(parts) > 1 {
		host = parts[0]
	}

	// TODO (sgarrity 12/23): I'm not convinced this is the best way to do this, but
	// we allow loopback to work for local testing (specifically, so that we can run envtest
	// against an httptest server). This won't change our actual security profile today
	// (since even if an external request is forged with this host header, it will fail the
	// multitenant middleware check for tenant name), but it's still not ideal.
	if net.ParseIP(host) != nil && host != "127.0.0.1" {
		uclog.Verbosef(ctx, "got request with IP address (%s) in host header, blocking", r.Host)
		uclog.IncrementEventWithPayload(ctx, eventRequestBlockedByIPInHostHeader, r.Host)
		status.Code = eventRequestBlockedByIPInHostHeader
		return status, ucerr.New("invalid host header (ip)")
	}

	return status, nil
}

// IsCallBlocked checks if call on particular user account are blocked
func (s *securityChecker) IsCallBlocked(ctx context.Context, username string) bool {
	// Store the username in the security context
	status := GetSecurityStatus(ctx)
	// If we are processing a call without a security context - abort
	if status == nil {
		uclog.IncrementEventWithPayload(ctx, eventRequestBlockedByUserName, username)
		return true
	}
	status.Username = username

	// Check if blocked
	if s.checkForBlockedUser(username) {
		uclog.IncrementEventWithPayload(ctx, eventRequestBlockedByUserName, username)
		return true
	}
	return false
}

func (s *securityChecker) FinishedRequest(status Status, r *http.Request, success bool) {

	for _, ip := range status.IPs {
		s.recordOutcomeForIP(r.Context(), ip, success)
	}
	if status.Username != "" {
		s.recordOutcomeForUser(r.Context(), status.Username, success)
	}
}

func getIPs(req *http.Request) ([]string, error) {
	var ips []string
	var lbIP string

	// TODO only do this in AWS, check if there a load balancer address in the headers
	if loadBalIP := req.Header.Get(headers.XRealIP); loadBalIP != "" {
		if net.ParseIP(loadBalIP) != nil {
			lbIP = strings.ReplaceAll(loadBalIP, " ", "")
		}
	}

	ip, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return ips, ucerr.New("Couldn't split ip and port for the remote address for the request")
	}
	if ip != "127.0.0.1" && ip != lbIP {
		ips = append(ips, ip)
	}

	// This is supposed to be set by a non-anonymous proxy but can also be spoofed by the client
	// or not set by an anonymous/elite proxy
	if forwardIPsStr := req.Header.Get(headers.XForwardedFor); forwardIPsStr != "" {
		forwardIPsStr = strings.ReplaceAll(forwardIPsStr, " ", "")
		forwardIPs := strings.SplitSeq(forwardIPsStr, ",")

		for forwardIP := range forwardIPs {
			if forwardIP != "127.0.0.1" && forwardIP != lbIP {
				ips = append(ips, forwardIP)
			}
		}
	}

	return ips, nil
}

func (s *securityChecker) checkForBlockedIP(ip string) bool {
	now := time.Now().UTC()
	s.ipMutex.Lock()
	defer s.ipMutex.Unlock()

	// Check if there is a local record for a particular ip
	r, ok := s.ipHistory[ip]
	if !ok {
		r = &callHistory{}
		s.ipHistory[ip] = r
	}

	if r.blockedUntil.After(now) {
		// The IP is currently blocked and continues to make calls, increase the penalty
		r.blockedUntil = r.blockedUntil.Add(penaltyPerBlockedCall)
		return true
	} else if !r.blockedUntil.IsZero() {
		// First call since the block expired resets the count
		r.blockedUntil = time.Time{}
		r.count = 1
	}

	return false
}

func (s *securityChecker) recordOutcomeForIP(ctx context.Context, ip string, success bool) {
	now := time.Now().UTC()

	s.ipMutex.Lock()
	defer s.ipMutex.Unlock()
	// Check if there is a record for a particular ip (could have expired during the call)
	r, ok := s.ipHistory[ip]
	if !ok {
		r = &callHistory{}
		s.ipHistory[ip] = r
	}
	// Update record for the given IP
	r.lastActive = now
	if success {
		if r.count > 0 {
			r.count--
		}
	} else {
		r.count++

		if r.blockedUntil.After(now) {
			// The IP is currently blocked and continues to make calls, increase the penalty
			// (can only get here if another thread failed during our call)
			r.blockedUntil = r.blockedUntil.Add(penaltyPerBlockedCall)
		} else if r.count > callFailureLimitPerIP {
			// First call triggering the block, starting with the smallest penalty
			r.blockedUntil = now.Add(penaltyPerBlockedCall)
		}

		uclog.IncrementEventWithPayload(ctx, eventFailureByIP, ip)
	}
}

func (s *securityChecker) checkForBlockedUser(user string) bool {
	now := time.Now().UTC()
	s.userMutex.Lock()
	defer s.userMutex.Unlock()

	// Check if there is a call record for a particular user
	r, ok := s.userHistory[user]
	if !ok {
		r = &callHistory{}
		s.userHistory[user] = r
	}
	r.count++
	r.lastActive = now
	if r.blockedUntil.After(now) {
		// The username is currently blocked and continues to make calls, increase the penalty
		r.blockedUntil = r.blockedUntil.Add(penaltyPerBlockedCall)
		return true
	} else if !r.blockedUntil.IsZero() {
		// First call since the block expired resets the count
		r.blockedUntil = time.Time{}
		r.count = 1
	}
	return false
}

func (s *securityChecker) recordOutcomeForUser(ctx context.Context, username string, success bool) {
	now := time.Now().UTC()

	s.userMutex.Lock()
	defer s.userMutex.Unlock()
	// Check if there is a record for a particular user (could have expired during the call)
	r, ok := s.userHistory[username]
	if !ok {
		r = &callHistory{}
		s.userHistory[username] = r
	}
	// Update record for the given user
	r.lastActive = now
	if success {
		if r.count > 0 {
			r.count--
		}
	} else {
		r.count++
		if r.blockedUntil.After(now) {
			// The user is currently blocked and continues to make calls, increase the penalty
			// (can only get here if another thread failed during our call)
			r.blockedUntil = r.blockedUntil.Add(penaltyPerBlockedCall)
		} else if r.count > callFailureLimitPerUser {
			// First call triggering the block, starting with the smallest penalty
			r.blockedUntil = now.Add(penaltyPerBlockedCall)
		}
		uclog.IncrementEventWithPayload(ctx, eventFailureByUserName, username)
	}
}

func (s *securityChecker) expireIPHistory() {
	now := time.Now().UTC()
	s.ipMutex.Lock()
	defer s.ipMutex.Unlock()
	for ip, r := range s.ipHistory {
		// Don't expire currently blocked records
		if r.blockedUntil.After(now) {
			continue
		}
		// Expire records that have seen less than one call per penalty period
		if now.Sub(r.lastActive) < time.Duration(r.count)*expirationPerBlockedCall {
			delete(s.ipHistory, ip)
		}
	}
}

func (s *securityChecker) expireUserHistory() {
	now := time.Now().UTC()
	s.userMutex.Lock()
	defer s.userMutex.Unlock()
	for u, r := range s.userHistory {
		// Don't expire currently blocked records
		if r.blockedUntil.After(now) {
			continue
		}
		// Expire records that have seen less than one call per penalty period
		if now.Sub(r.lastActive) < time.Duration(r.count)*expirationPerBlockedUser {
			delete(s.userHistory, u)
		}
	}
}
