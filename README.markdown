# Chimney

Collects smoketest results and funnels them to graphite.

Chimney will poll a set of URLs that correspond to applications
running [django-smoketest](https://github.com/ccnmtl/django-smoketest)
(or something that outputs results in the same format), collects their
results and submits them to a central Graphite server. Pretty
straightforward.

## Running

    $ chimney -config=/path/to/chimney/config.json

`Ctrl-C` to stop it. You probably want to keep it running with
something like upstart or supervisord, though.

## Configuration

Here's a sample config file:

     {
         "GraphiteBase": "graphite.example.com:2003",
         "PollInterval": 300,
         "Jitter": 20,
     
         "Tests": [
             {
                 "Url": "http://app1.example.com/smoketest/",
                 "MetricPrefix": "smoketest.app1."
             },
             {
                 "Url": "http://app2.example.com/smoketest/",
                 "MetricPrefix": "smoketest.app2."
             },
             {
                 "Url": "http://app3.example.com/smoketest/",
                 "MetricPrefix": "smoketest.app3."
             }
         ]
    }

It's just JSON. Mind your trailing commas though.

We tell chimney where the Graphite/Carbon server is for it to send
results to. Then we tell it to poll each test roughly every five
minutes (with 20 seconds of random jitter to avoid thundering
herds). We tell it about three different applications that have
smoketest URLs. Each just needs to have the URL for the smoketests and
a prefix for graphite.

Chimney will spawn a thread for each test (goroutine, actually) that,
once per interval specified, makes a `GET` request to the test URL
(with `Accept: application/json`), parses the results, and sends them
along to Graphite under the appropriate label.

A typical response from a smoketest might look something like this:

    {"status": "FAIL",
     "tests_failed": 1,
     "errored_tests": ["app1.smoke.Test.test_connectivity"],
     "tests_run": 19,
     "test_classes": 10,
     "tests_passed": 17,
     "time": 1298.7720966339111,
     "failed_tests": ["app1.main.smoke.Test.test_watchdir"],
     "tests_errored": 1}

Chimney will turn that into Graphite metrics like (assuming this is
'app1' per the configuration above):

    smoketest.app1.run 19
    smoketest.app1.passed 17
    smoketest.app1.classes 10
    smoketest.app1.failed 1
    smoketest.app1.errored 1
    smoketest.app1.time 1298.7720966339111

## Docker

There's also a docker packaged version of chimney available. Run it
like:

    $ docker run -v /path/to/config.json:/etc/chimney/config.json ccnmtl/chimney
