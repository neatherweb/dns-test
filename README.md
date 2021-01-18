# dns-test

Command tool written in Go to support testing of DNS servers.

The tool was written to load test vendor DNS solutions in a cloud environment and to assist in baselining records and RTT for pre/post migration.

## Usage

```
Usage: dns-test
  -clients int
        Number of client Go Routines to spawn (default 1)
  -debug
        Enable debug messages
  -delay int
        Delay between client requests in ms. Use 0 for load test - with CAUTION not to DOS your prod systems! (default 100)
  -duration int
        How long to run the test for in ms. Negative number will result in one iteration through the test inputs. (default -1)
  -input string
        CSV input file for queries, rows formatted as <type>,<name_to_query>,<[expected_answer0..n]>
  -json
        Ouput in json format
  -server string
        DNS server to target - format '10.1.1.1:53'
  -validate
        Validate the 1st response matches the expected set of responses.
Output:
        The test output will show raw stats per client and overall, followed by
        summary data.

        Errors: (count) the server did not respond or there was a unexpected error.
        Noanswer: (count) the server response was recieved by there was no answers to the query.
        Success: (count) the server response had answers
        Verified: (count) the server response was validated against the expected result set
        Incorrect: (count) the server response was not found in the expected result set
        QPS: queries per second for the test run
        RTT-min: minimum round trip time observed
        RTT-max: maximum round trip time observed
        RTT-avg: average round trip time observed
        Test duration(ms): overall test duration
```

## Downloading

Binaries are available from [releases](https://github.com/neatherweb/dns-test/releases/latest)

## Examples

Single iteration of the input list and validate results.

```shell
[admin@testserver ~]$ ./dns-test -input inputs.csv -server 192.168.1.14:53 -validate
Client Results 
[{0 0 3 0 0 0 0 3 0}]
Overall Results 
{0 0 3 0 0 0 0 3 0}
--------------------------------------------------------
Result Summary
--------------------------------------------------------
  Errors: 0
  Noanswer: 0
  Success: 3
  Verified: 3
  Incorrect: 0
  QPS: 9.947639
  RTT-min: 0
  RTT-max: 0
  RTT-avg: 0
  Test duration(ms): 301
```

Load test for 30 seconds with 20 clients as fast as you can (delay 0).

```shell
[admin@testserver ~]$ ./dns-test -input inputs.csv -server 192.168.1.14:53 -duration 30000 -clients 20 -delay 0
Client Results 
[{0 0 135553 602 0 12 0 0 0} {0 0 135913 537 0 12 0 0 0} {0 0 135537 606 0 12 0 0 0} {0 0 136019 579 0 12 0 0 0} {0 0 135852 579 0 12 0 0 0} {0 0 135694 593 0 12 0 0 0} {0 0 135680 613 0 12 0 0 0} {0 0 136094 573 0 12 0 0 0} {0 0 135892 610 0 12 0 0 0} {0 0 135790 587 0 12 0 0 0} {0 0 135951 577 0 12 0 0 0} {0 0 135821 551 0 12 0 0 0} {0 0 135856 573 0 12 0 0 0} {0 0 135634 617 0 12 0 0 0} {0 0 135883 584 0 16 0 0 0} {0 0 135632 587 0 12 0 0 0} {0 0 135607 628 0 12 0 0 0} {0 0 135764 577 0 12 0 0 0} {0 0 135722 561 0 12 0 0 0} {0 0 135640 612 0 12 0 0 0}]
Overall Results 
{0 0 2715534 11746 0 16 0 0 0}
--------------------------------------------------------
Result Summary
--------------------------------------------------------
  Errors: 0
  Noanswer: 0
  Success: 2715534
  Verified: 0
  Incorrect: 0
  QPS: 90517.454106
  RTT-min: 0
  RTT-max: 16
  RTT-avg: 0
  Test duration(ms): 30000
```

## Building

GOOS=linux GOARCH=amd64 go build -o bin/dns-test

GOOS=windows GOARCH=amd64 go build -o bin/dns-test.exe
