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
