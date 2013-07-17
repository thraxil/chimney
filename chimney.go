package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"time"
)

type SmoketestData struct {
	Url          string
	MetricPrefix string
}

func (s *SmoketestData) Monitor(graphite string, interval int, jitter int) {
	for {
		j := rand.Intn(jitter)
		time.Sleep(time.Duration(interval+j) * time.Second)
		s.Check(graphite)
	}
}

/* Example:
{"status": "FAIL",
 "tests_failed": 1,
 "errored_tests": ["wardenclyffe.cuit.smoke.CUITSFTPTest.test_connectivity"],
 "tests_run": 19,
 "test_classes": 10,
 "tests_passed": 17,
 "time": 1298.7720966339111,
 "failed_tests": ["wardenclyffe.main.smoke.WatchDirTest.test_watchdir"],
 "tests_errored": 1}
*/

type TestResult struct {
	Status       string   `json:"status"`
	TestsPassed  int      `json:"tests_passed"`
	TestsFailed  int      `json:"tests_failed"`
	TestsErrored int      `json:"tests_errored"`
	TestsRun     int      `json:"tests_run"`
	TestClasses  int      `json:"test_classes"`
	Time         float64  `json:"time"`
	ErroredTests []string `json:"errored_tests"`
	FailedTests  []string `json:"failed_tests"`
}

func (t *TestResult) ToBytes(metric_prefix string) []byte {
	now := int32(time.Now().Unix())
	buffer := bytes.NewBufferString("")
	fmt.Fprintf(buffer, "%srun %d %d\n", metric_prefix, t.TestsRun, now)
	fmt.Fprintf(buffer, "%spassed %d %d\n", metric_prefix, t.TestsPassed, now)
	fmt.Fprintf(buffer, "%sfailed %d %d\n", metric_prefix, t.TestsFailed, now)
	fmt.Fprintf(buffer, "%serrored %d %d\n", metric_prefix, t.TestsErrored, now)
	fmt.Fprintf(buffer, "%sclasses %d %d\n", metric_prefix, t.TestClasses, now)
	fmt.Fprintf(buffer, "%stime %f %d\n", metric_prefix, t.Time, now)
	return buffer.Bytes()
}

func (s *SmoketestData) Check(graphite string) {
	var clientGraphite net.Conn
	clientGraphite, err := net.Dial("tcp", graphite)
	if err != nil || clientGraphite == nil {
		return
	}
	defer clientGraphite.Close()

	// go doesn't like our certificates:
	// "x509: certificate signed by unknown authority"
	// Ideally, we'd whitelist them, but for smoketest
	// purposes, it's probably OK to just go insecure
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: transport}
	req, err := http.NewRequest("GET", s.Url, nil)
	if err != nil {
		return
	}
	req.Header.Set("accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	tr := TestResult{}
	err = json.Unmarshal(body, &tr)
	if err != nil {
		return
	}

	clientGraphite.Write(tr.ToBytes(s.MetricPrefix))
}

type ConfigData struct {
	GraphiteBase string
	PollInterval int
	Jitter       int
	Tests        []SmoketestData
}

func main() {
	var configFile string
	flag.StringVar(&configFile, "config", "/etc/chimney/config.json", "JSON config file")
	flag.Parse()
	rand.Seed(int64(time.Now().Unix()))

	file, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Fatal(err)
	}

	f := ConfigData{}
	err = json.Unmarshal(file, &f)
	if err != nil {
		log.Fatal(err)
	}

	dummy := make(chan int)
	for _, t := range f.Tests {
		test := t
		go test.Monitor(f.GraphiteBase, f.PollInterval, f.Jitter)
	}
	// wait on a dummy channel. equiv of "while(True)"
	// Ie, this program runs until it gets a Ctrl-C or equivalent signal
	<-dummy
}
