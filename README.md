# interview_idea
An idea I am floating for interviews

Install redis and run it.
```bash
$ brew install redis
$redis-server /usr/local/etc/redis.conf
```

Run the application (see -h for help and optional parameters)
```bash
$ go run main.go
```

You will be directed to go to a url (in the main.go log). Instructions will be there. 

# Docker

Start up using docker-compose
```
docker-compose up
```

Go to http://$DOCKER_HOST:9090
