package main

import (
	"bytes"
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
	fmt.Println("starting monitor for", s.Url)
	rand.Seed(int64(time.Now().Unix()))
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

func (s *SmoketestData) Check(graphite string) {
	var clientGraphite net.Conn
	clientGraphite, err := net.Dial("tcp", graphite)
	if err != nil || clientGraphite == nil {
		return
	}
	defer clientGraphite.Close()

	client := &http.Client{}
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

	now := int32(time.Now().Unix())
	buffer := bytes.NewBufferString("")
	fmt.Fprintf(buffer, "%srun %d %d\n", s.MetricPrefix, tr.TestsRun, now)
	fmt.Fprintf(buffer, "%spassed %d %d\n", s.MetricPrefix, tr.TestsPassed, now)
	fmt.Fprintf(buffer, "%sfailed %d %d\n", s.MetricPrefix, tr.TestsFailed, now)
	fmt.Fprintf(buffer, "%serrored %d %d\n", s.MetricPrefix, tr.TestsErrored, now)
	fmt.Fprintf(buffer, "%sclasses %d %d\n", s.MetricPrefix, tr.TestClasses, now)
	fmt.Fprintf(buffer, "%stime %f %d\n", s.MetricPrefix, tr.Time, now)
	fmt.Print(string(buffer.Bytes()))
	clientGraphite.Write(buffer.Bytes())

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
