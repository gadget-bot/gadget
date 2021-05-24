package network_utils

// Most of this package is based on the fine work at https://github.com/mehrdadrad/mylg

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/gadget-bot/gadget/router"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

type HTTPPing struct {
	url           string
	host          string
	interval      time.Duration
	timeout       time.Duration
	count         int
	method        string
	uAgent        string
	buf           string
	transport     *http.Transport
	rAddr         net.Addr
	nsTime        time.Duration
	quiet         bool
	dCompress     bool
	kAlive        bool
	TLSSkipVerify bool
	tracerEnabled bool
	ipv4          bool
	ipv6          bool
}

type HTTPResult struct {
	StatusCode int
	TotalTime  float64
	Size       int
	Proto      string
	Server     string
	Status     string
	Trace      HTTPTrace
}

type HTTPTrace struct {
	ConnectionTime  float64
	TimeToFirstByte float64
}

func (p HTTPPing) IPVersion(t string) string {
	if p.ipv4 {
		return fmt.Sprintf("%s4", t)
	} else if p.ipv6 {
		return fmt.Sprintf("%s6", t)
	}

	return t
}

func (p *HTTPPing) run() (map[int]float64, []float64, error) {
	if p.method != "GET" && p.method != "POST" && p.method != "HEAD" {
		return nil, nil, fmt.Errorf("Error: Method '%s' not recognized.", p.method)
	}

	var (
		results       = make(map[int]float64, 10)
		responseTimes []float64
	)

	for i := 0; i < p.count; i++ {
		if r, err := p.Ping(); err == nil {
			results[r.StatusCode]++
			responseTimes = append(responseTimes, r.TotalTime*1e3)
		} else {
			results[-1]++
		}
		time.Sleep(p.interval)
	}
	return results, responseTimes, nil
}

func (p *HTTPPing) setTransport() {
	p.transport = &http.Transport{
		DisableKeepAlives:  !p.kAlive,
		DisableCompression: p.dCompress,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: p.TLSSkipVerify,
		},
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return net.Dial(p.IPVersion("tcp"), addr)
		},
	}
}

func (p *HTTPPing) Ping() (HTTPResult, error) {
	var (
		r     HTTPResult
		sTime time.Time
		resp  *http.Response
		req   *http.Request
		err   error
	)

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Don't follow redirects
			return http.ErrUseLastResponse
		},
		Timeout:   p.timeout,
		Transport: p.transport,
	}

	sTime = time.Now()

	if p.method == "POST" {
		r.Size = len(p.buf)
		reader := strings.NewReader(p.buf)
		req, err = http.NewRequest(p.method, p.url, reader)
	} else {
		req, err = http.NewRequest(p.method, p.url, nil)
	}

	if err != nil {
		return r, err
	}

	req.Header.Add("User-Agent", p.uAgent)
	if p.tracerEnabled && !p.quiet {
		req = req.WithContext(httptrace.WithClientTrace(req.Context(), tracer(&r)))
	}
	resp, err = client.Do(req)

	if err != nil {
		return r, err
	}
	defer resp.Body.Close()

	r.TotalTime = time.Since(sTime).Seconds()

	if p.method == "GET" {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return r, err
		}
		r.Size = len(body)
	} else {
		io.Copy(ioutil.Discard, resp.Body)
	}

	r.StatusCode = resp.StatusCode
	r.Proto = resp.Proto
	return r, nil
}

func tracer(r *HTTPResult) *httptrace.ClientTrace {
	var (
		begin   = time.Now()
		elapsed time.Duration
	)

	return &httptrace.ClientTrace{
		ConnectDone: func(network, addr string, err error) {
			elapsed = time.Since(begin)
			begin = time.Now()
			r.Trace.ConnectionTime = elapsed.Seconds() * 1e3
		},
		GotFirstResponseByte: func() {
			elapsed = time.Since(begin)
			begin = time.Now()
			r.Trace.TimeToFirstByte = elapsed.Seconds() * 1e3
		},
	}
}

func calcStats(c map[int]float64, s []float64) map[string]float64 {
	var r = make(map[string]float64, 5)

	for k, v := range c {
		if k < 0 {
			continue
		}
		r["sum"] += v
	}

	for _, v := range s {
		// maximum
		if r["max"] < v {
			r["max"] = v
		}
		// minimum
		if r["min"] > v || r["min"] == 0 {
			r["min"] = v
		}
		// average
		if r["avg"] == 0 {
			r["avg"] = v
		} else {
			r["avg"] = (r["avg"] + v) / 2
		}
	}
	return r
}

func printStats(p HTTPPing, c map[int]float64, s []float64) string {
	var statsOutput string

	r := calcStats(c, s)

	totalReq := r["sum"] + c[-1]
	failPct := 100 - (100*r["sum"])/totalReq

	statsOutput = fmt.Sprintf("*%s HTTP ping statistics*\n", p.host)
	statsOutput += fmt.Sprintf("- %.0f %s requests transmitted, %.0f replies received, %.0f%% requests failed\n", totalReq, p.method, r["sum"], failPct)
	statsOutput += fmt.Sprintf("- HTTP Round-trip min/avg/max = %.2f/%.2f/%.2f ms\n", r["min"], r["avg"], r["max"])

	return statsOutput
}

func newPing(method string, URL string, count int, interval string, tlsVerify bool) (*HTTPPing, error) {
	u, err := url.Parse(URL)
	if err != nil {
		return &HTTPPing{}, fmt.Errorf("Unparseable url")
	}

	host, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		host = u.Host
	}

	sTime := time.Now()

	p := &HTTPPing{
		url:           URL,
		host:          u.Host,
		count:         count,
		tracerEnabled: false,
		uAgent:        "Gadget (https://github.com/gadget-bot/gadget)",
		dCompress:     false,
		kAlive:        false,
		TLSSkipVerify: !tlsVerify,
		ipv4:          false,
		ipv6:          false,
		nsTime:        time.Since(sTime),
	}

	ipAddr, err := net.ResolveIPAddr(p.IPVersion("ip"), host)
	if err != nil {
		return &HTTPPing{}, fmt.Errorf("cannot resolve %s: Unknown host", host)
	}

	p.rAddr = ipAddr
	p.interval, err = time.ParseDuration(interval)
	if err != nil {
		return p, fmt.Errorf("Failed to parse interval: %s. Correct syntax is <number>s/ms", err)
	}
	// set timeout
	timeout := "10s"
	p.timeout, err = time.ParseDuration(timeout)
	if err != nil {
		return p, fmt.Errorf("Failed to parse timeout: %s. Correct syntax is <number>s/ms", err)
	}
	// set method
	p.method = method
	p.method = strings.ToUpper(p.method)

	p.setTransport()

	return p, nil
}

func runHTTPPing() *router.MentionRoute {
	var pluginRoute router.MentionRoute
	pluginRoute.Permissions = append(pluginRoute.Permissions, "*")
	pluginRoute.Name = "network_utils.runHTTPPing"
	pluginRoute.Pattern = `(?i)^hping( (get|post|head))? <?(https?://[^\s>]+)>?( ([0-9]+)( ([0-9]+(s|ms)))?)?$`
	pluginRoute.Description = "Sends HTTP Pings to a given URL"
	pluginRoute.Help = "hping [get|post|head] URL [COUNT] [INTERVAL(s|ms)]"
	pluginRoute.Plugin = func(api slack.Client, router router.Router, ev slackevents.AppMentionEvent, message string) {
		msgRef := slack.NewRefToMessage(ev.Channel, ev.TimeStamp)
		api.AddReaction("male-detective", msgRef)

		re := regexp.MustCompile(pluginRoute.Pattern)
		results := re.FindStringSubmatch(message)
		method := results[2]
		checkUrl := results[3]
		count := results[5]
		interval := results[7]

		if method == "" {
			method = "get"
		}

		if count == "" {
			count = "3"
		}

		if interval == "" {
			interval = "2s"
		}

		var countInt int
		fmt.Sscan(count, &countInt)

		// Create a new HTTPing
		pinger, _ := newPing(method, checkUrl, countInt, interval, false)

		responses, responseTimes, _ := pinger.run()

		// Here's how we send a reply
		api.PostMessage(
			ev.Channel,
			slack.MsgOptionText(
				printStats(*pinger, responses, responseTimes),
				false,
			),
		)
	}

	// We've got to return the MentionRoute
	return &pluginRoute
}

// This function is used to retrieve all Mention Routes from this plugin
func GetMentionRoutes() []router.MentionRoute {
	return []router.MentionRoute{
		*runHTTPPing(),
		*queryWhois(),
	}
}
