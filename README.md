# interview_idea
An idea I am floating for interviews

Install redis and run it.
```bash
$ brew install redis
$ redis-server /usr/local/etc/redis.conf
```

Run the application (see -h for help and optional parameters)
```bash
$ go run main.go
```

The binary will direct you to go to a url (in the main.go log). Instructions for the candidate will be there.

# Instructions for the interviewer
After the candidate has time to work on a solution, we can dig deeper with the following:

## Lite Version
    - The candidate has tracked valid and invalid calls
        - how would we structure logging?
        - how would we structure monitoring?
        - how would we structure alerting?
        - We want to scale this out to 1000s of machines. How does any previous answer change?
        - How would does the candidate imagine the workflow to work with 1000s of machines? What about network failures?

## Full Version

    - Let the candidate know that our api verification is slow and they will see many duplicates in the requests. How can they mitigate this?
    - Let the candidate know that the verification boxes are getting bogged down and are becoming slow. How can they maximize through out?
    - Let them know that the system is a bit unstable, and some requests are being dropped with 500 level messages. How can we mitigate this?
    - Assume that the input is from a very large file, what would they change? What if it is a network stream?
    - What logging, monitoring, and metrics would the candidate consider for their service?
    - Talk to the candidate about memory vs disk usage trade offs they have made. What is the big-O notation of their methods?
    - ...

Also, for the interviewer, take note of the options on the interview service by running `go run main.go -h`. You can have the program show solutions, add instability, increase string lengths and batch sizes. You could have two services running, one simple and one complex. Your call :)


