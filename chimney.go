package main // import "github.com/thraxil/chimney"

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"

	"math/rand"
	"net"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/kelseyhightower/envconfig"
)

type config struct {
	LogLevel string `envconfig:"LOG_LEVEL"`
}

type smoketestData struct {
	URL          string
	MetricPrefix string
}

func (s *smoketestData) Monitor(graphite string, interval int, jitter int) {
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

type testResult struct {
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

func (t *testResult) ToBytes(metricPrefix string) []byte {
	now := int32(time.Now().Unix())
	buffer := bytes.NewBufferString("")
	fmt.Fprintf(buffer, "%srun %d %d\n", metricPrefix, t.TestsRun, now)
	fmt.Fprintf(buffer, "%spassed %d %d\n", metricPrefix, t.TestsPassed, now)
	fmt.Fprintf(buffer, "%sfailed %d %d\n", metricPrefix, t.TestsFailed, now)
	fmt.Fprintf(buffer, "%serrored %d %d\n", metricPrefix, t.TestsErrored, now)
	fmt.Fprintf(buffer, "%sclasses %d %d\n", metricPrefix, t.TestClasses, now)
	fmt.Fprintf(buffer, "%stime %f %d\n", metricPrefix, t.Time, now)
	return buffer.Bytes()
}

func (s *smoketestData) Check(graphite string) {
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
	req, err := http.NewRequest("GET", s.URL, nil)
	if err != nil {
		tr := testResult{Status: "FAIL", TestsPassed: 0, TestsFailed: 0, TestsErrored: 0, TestsRun: 0, TestClasses: 0, Time: 0.0}
		clientGraphite.Write(tr.ToBytes(s.MetricPrefix))
		return
	}
	req.Header.Set("accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		tr := testResult{Status: "FAIL", TestsPassed: 0, TestsFailed: 0, TestsErrored: 0, TestsRun: 0, TestClasses: 0, Time: 0.0}
		clientGraphite.Write(tr.ToBytes(s.MetricPrefix))
		return
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		tr := testResult{Status: "FAIL", TestsPassed: 0, TestsFailed: 0, TestsErrored: 0, TestsRun: 0, TestClasses: 0, Time: 0.0}
		clientGraphite.Write(tr.ToBytes(s.MetricPrefix))
		return
	}

	tr := testResult{}
	err = json.Unmarshal(body, &tr)
	if err != nil {
		tr := testResult{Status: "FAIL", TestsPassed: 0, TestsFailed: 0, TestsErrored: 0, TestsRun: 0, TestClasses: 0, Time: 0.0}
		clientGraphite.Write(tr.ToBytes(s.MetricPrefix))
		return
	}

	clientGraphite.Write(tr.ToBytes(s.MetricPrefix))
}

type configData struct {
	GraphiteBase string
	PollInterval int
	Jitter       int
	Tests        []smoketestData
}

func main() {
	log.SetLevel(log.InfoLevel)
	var c config
	err := envconfig.Process("chimney", &c)
	if err != nil {
		log.Fatal(err.Error())
	}
	if c.LogLevel == "DEBUG" {
		log.SetLevel(log.DebugLevel)
	}
	if c.LogLevel == "WARN" {
		log.SetLevel(log.WarnLevel)
	}
	if c.LogLevel == "ERROR" {
		log.SetLevel(log.ErrorLevel)
	}
	if c.LogLevel == "FATAL" {
		log.SetLevel(log.FatalLevel)
	}
	var configFile string
	flag.StringVar(&configFile, "config", "/etc/chimney/config.json", "JSON config file")
	flag.Parse()
	rand.Seed(int64(time.Now().Unix()))

	file, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.WithFields(
			log.Fields{
				"error":      err,
				"configFile": configFile,
			},
		).Fatal("couldn't read config file")
	}

	f := configData{}
	err = json.Unmarshal(file, &f)
	if err != nil {
		log.WithFields(
			log.Fields{
				"error": err,
			},
		).Fatal("couldn't parse config file")
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
