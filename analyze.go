package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/intel/tfortools"
	"github.com/pkg/errors"
)

const (
	reportTmpl = `{{.MyName}} {{.MyVersion}}/j{{.Jobs}} by {{.Author}}

Reporting by: {{.Org}} — {{.Email}}
From {{.DateBegin}} to {{.DateEnd}}

Domain: {{.Domain}}
Domain RUA Email: {{.DomainRUA}}
Policy: p={{.Disposition}}; dkim={{.DKIM}}; spf={{.SPF}}

Reports({{.Count}}):
`

	rowTmpl = `{{ table (sort . %s)}}`

	reportTmplJSON = `{
	"apiVersion": "v1",
	"status": "success",
	"processorMeta": {
		"applicationName": "{{.MyName}}",
		"jobs": "{{.Jobs}}",
		"processorVersion": "{{.MyVersion}}"
	},
	"reportCount": "{{.Count}}",
	"reports": [
		{
			"reportingOrg": "{{.Org}}",
			"reportingEmail": "{{.Email}}",
			"reportStartDate": "{{.DateBegin}}",
			"reportEndDate": "{{.DateEnd}}",
			"reportedDomain": "{{.Domain}}",
			"reportedDomainRUA": "{{.DomainRUA}}",
			"reportedPolicy": {
				"disposition": "{{.Disposition}}",
				"dkim": "{{.DKIM}}",
				"spf": "{{.SPF}}"
			},

			"entries":
			`

	rowTmplJSON = `{{.}}
		}
	]
}`
)

// My template vars
type headVars struct {
	MyName      string
	MyVersion   string
	Jobs        string
	Author      string
	Org         string
	Email       string
	DateBegin   string
	DateEnd     string
	Domain      string
	DomainRUA   string
	Disposition string
	DKIM        string
	SPF         string
	Pct         int
	Count       int
}

// Entry representes a single entry
type Entry struct {
	IP    string
	Count int
	From  string
	RFrom string
	RDKIM string
	RSPF  string
}

type IP struct {
	IP   string
	Name string
}

func getDomainRUA(domain string) (string) {
	fmt.Println("Pulling live DMARC record for domain " + domain)

	txtrecords, _ := net.LookupTXT("_dmarc." + domain)

	var record string

	for _, txt := range txtrecords {
		fmt.Println("Pulled live DMARC record: " + txt)
		record = txt
	}

	s := strings.Split(record, "; ")

	r := s[len(s)-1]

	ns := strings.Replace(r, "rua=mailto:", "", -1)

	fmt.Println("Processed live DMARC RUA Email: " + ns)

	return ns
}

func ParallelSolve(ctx *Context, iplist []IP) []IP {
	verbose("ParallelSolve with %d workers", ctx.jobs)

	var lock sync.Mutex

	wg := &sync.WaitGroup{}
	queue := make(chan IP, ctx.jobs)

	resolved := iplist

	ind := 0

	for i := 0; i < ctx.jobs; i++ {
		wg.Add(1)

		debug("setting up w%d", i)
		go func(n int, wg *sync.WaitGroup) {
			defer wg.Done()

			var name string

			for e := range queue {
				ips, err := ctx.r.LookupAddr(e.IP)
				if err != nil {
					name = e.IP
				} else {
					name = ips[0]
				}

				lock.Lock()
				resolved[ind].Name = name
				lock.Unlock()
				debug("w%d - ip=%s - name=%s", n, e.IP, name)
			}
		}(i, wg)
	}

	for _, q := range iplist {
		queue <- q
	}

	close(queue)
	wg.Wait()

	return resolved
}

// GatherRows extracts all IP and return the rows
func GatherRows(ctx *Context, r Feedback) []Entry {
	var (
		rows    []Entry
		iplist  []IP
		newlist []IP
	)

	ipslen := len(r.Records)

	if fNoResolv {
		ctx.r = NullResolver{}
	}

	verbose("Resolving all %d IPs", ipslen)
	iplist = make([]IP, ipslen)
	// Get all IPs
	for i, report := range r.Records {
		iplist[i] = IP{IP: report.Row.SourceIP.String()}
	}

	// Now we have a nice array
	newlist = ParallelSolve(ctx, iplist)
	verbose("Resolved %d IPs", ipslen)

	for i, report := range r.Records {
		ip0 := newlist[i].Name
		current := Entry{
			IP:    ip0,
			Count: report.Row.Count,
			From:  report.Identifiers.HeaderFrom,
			RSPF:  report.AuthResults.SPF.Result,
			RDKIM: report.AuthResults.DKIM.Result,
		}
		if report.AuthResults.DKIM.Domain == "" {
			current.RFrom = report.AuthResults.SPF.Domain
		} else {
			current.RFrom = report.AuthResults.DKIM.Domain
		}
		rows = append(rows, current)
	}
	return rows
}

// Analyze extract and display what we want
func Analyze(ctx *Context, r Feedback) (string, error) {
	var buf bytes.Buffer

	tmplvars := &headVars{
		MyName:      MyName,
		MyVersion:   MyVersion,
		Jobs:        fmt.Sprintf("%d", fJobs),
		Author:      Author,
		Org:         r.Metadata.OrgName,
		Email:       r.Metadata.Email,
		DateBegin:   time.Unix(r.Metadata.Date.Begin, 0).String(),
		DateEnd:     time.Unix(r.Metadata.Date.End, 0).String(),
		Domain:      r.Policy.Domain,
		DomainRUA:   getDomainRUA(r.Policy.Domain),
		Disposition: r.Policy.P,
		DKIM:        r.Policy.ADKIM,
		SPF:         r.Policy.ASPF,
		Pct:         r.Policy.Pct,
		Count:       len(r.Records),
	}

	rows := GatherRows(ctx, r)
	if len(rows) == 0 {
		return "", fmt.Errorf("empty report")
	}

	// Header
	t := template.Must(template.New("r").Parse(string(reportTmpl)))
	err := t.ExecuteTemplate(&buf, "r", tmplvars)
	if err != nil {
		return "", errors.Wrapf(err, "error in template 'r'")
	}

	// Generate our template
	sortTmpl := fmt.Sprintf(rowTmpl, fSort)
	err = tfortools.OutputToTemplate(&buf, "reports", sortTmpl, rows, nil)
	if err != nil {
		return "", errors.Wrapf(err, "error in template 'reports'")
	}

	return buf.String(), nil
}


// Analyze extract and display what we want
func AnalyzeJSON(ctx *Context, r Feedback) (string, error) {
	var buf bytes.Buffer

	tmplvars := &headVars{
		MyName:      MyName,
		MyVersion:   MyVersion,
		Jobs:        fmt.Sprintf("%d", fJobs),
		Author:      Author,
		Org:         r.Metadata.OrgName,
		Email:       r.Metadata.Email,
		DateBegin:   time.Unix(r.Metadata.Date.Begin, 0).String(),
		DateEnd:     time.Unix(r.Metadata.Date.End, 0).String(),
		Domain:      r.Policy.Domain,
		DomainRUA:   getDomainRUA(r.Policy.Domain),
		Disposition: r.Policy.P,
		DKIM:        r.Policy.ADKIM,
		SPF:         r.Policy.ASPF,
		Pct:         r.Policy.Pct,
		Count:       len(r.Records),
	}

	rows := GatherRows(ctx, r)
	if len(rows) == 0 {
		return "", fmt.Errorf("empty report")
	}	

	pagesJson, perr := json.MarshalIndent(rows, "\t\t\t", "\t")
    if perr != nil {
        fmt.Println("Cannot encode to JSON")
    }
	
	newReportJSON := strings.Split(string(pagesJson), "%!(EXTRA")

	//fmt.Println(newReportJSON[0])

	// Header
	t := template.Must(template.New("r").Parse(string(reportTmplJSON)))
	err := t.ExecuteTemplate(&buf, "r", tmplvars)
	if err != nil {
		return "", errors.Wrapf(err, "error in template 'r'")
	}

	// Generate our template
	sortTmpl := fmt.Sprintf(rowTmplJSON+"%s", fSort)
	err = tfortools.OutputToTemplate(&buf, "reports", sortTmpl, newReportJSON[0], nil)
	if err != nil {
		return "", errors.Wrapf(err, "error in template 'reports'")
	}

	formattedString := strings.Split(buf.String(), "%!(EXTRA")[0]
	roundTwoFormattedString := strings.Split(formattedString, `"Count" "dsc"`)[0]
	
	return roundTwoFormattedString, nil
}
