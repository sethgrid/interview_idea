package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"code.google.com/p/go-uuid/uuid"
	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/mux"
)

var Port int
var RandStringLen int
var MaxBatchSize int
var RedisAddr string

var AllAPIKeys map[string]bool

func init() {
	flag.IntVar(&Port, "port", 9090, "set the port as an integer")
	flag.IntVar(&RandStringLen, "string-length", 15, "set the random string length")
	flag.IntVar(&MaxBatchSize, "max-in-batch", 15, "max number of cases in each batch")
	flag.StringVar(&RedisAddr, "redis", ":6379", "set host and port for redis")
}

func main() {
	flag.Parse()
	r := mux.NewRouter()

	log.Printf("starting on 0.0.0.0:%d", Port)

	r.HandleFunc("/", rootHandler)
	r.HandleFunc("/generate", generateHandler)
	r.HandleFunc("/validate/apikey/{key}", validateAPIKeyHandler)
	r.HandleFunc("/validate/batch", validateBatchHandler)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", Port), r))
}

func generateHandler(w http.ResponseWriter, r *http.Request) {
	c := r.URL.Query().Get("count")
	if c == "" {
		c = "10"
	}
	count, err := strconv.Atoi(c)
	if err != nil {
		handleErr(w, http.StatusBadRequest, "count param should be a number")
		return
	}
	w.Write([]byte(genInput(count)))
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(fmt.Sprintf("use http://localhost:%d/generate?count=10\n", Port)))
}

func validateBatchHandler(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		handleErr(w, http.StatusInternalServerError, "unable to read post body")
		return
	}

	conn, err := redis.Dial("tcp", RedisAddr)
	if err != nil {
		log.Println("unable to connect to redis when validating batch", err.Error())
		handleErr(w, http.StatusInternalServerError, "unable to reach key store")
		return
	}

	lines := strings.Split(string(body), "\n")
	for i, line := range lines {
		if i == 0 || line == "" {
			continue
		}
		key := fmt.Sprintf("%s_%d", lines[0], i-1)

		resp, err := conn.Do("GET", key)
		if err != nil {
			log.Printf("error getting batch '%s' - %s", key, err.Error())
			handleErr(w, http.StatusInternalServerError, "unable to query key store")
			return
		}

		if resp == nil {
			handleErr(w, http.StatusNotFound, fmt.Sprintf("batch '%s' does not exist", key))
			return
		}
		data, err := redis.String(resp, err)
		if err != nil {
			log.Println("error getting string data ", err.Error())
			handleErr(w, http.StatusInternalServerError, "unable to read data")
			return
		}
		if data != line {
			w.Write([]byte(fmt.Sprintf("invalid submission %s_%d got %s, want %s", lines[0], i-1, line, data)))
		}
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok\n"))
}

func validateAPIKeyHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key, ok := vars["key"]
	if !ok {
		handleErr(w, http.StatusBadRequest, "must provide api key as URI segment")
		return
	}

	conn, err := redis.Dial("tcp", RedisAddr)
	if err != nil {
		log.Println("unable to connect to redis when validating apikey", err.Error())
		handleErr(w, http.StatusInternalServerError, "unable to reach key store")
		return
	}

	resp, err := conn.Do("GET", key)
	if err != nil {
		log.Printf("error getting api key '%s' - %s", key, err.Error())
		handleErr(w, http.StatusInternalServerError, "unable to query key store")
		return
	}

	if resp == nil {
		handleErr(w, http.StatusNotFound, fmt.Sprintf("apikey '%s' does not exist", key))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok\n"))
}

func handleErr(w http.ResponseWriter, statusCode int, msg string) {
	w.WriteHeader(statusCode)
	w.Write([]byte(msg + "\n"))
}

func genInput(count int) string {
	conn, err := redis.Dial("tcp", RedisAddr)
	if err != nil {
		log.Println("unable to connect to redis when generating input ", err.Error())
		return "<< error >>"
	}

	s := fmt.Sprintf("%d\n", count)
	for i := 1; i <= count; i++ {
		batchName := genBatchName()
		s += fmt.Sprintf("batch %s\n", batchName)

		for j := 0; j <= rand.Intn(MaxBatchSize); j++ {
			apiKey := genAPIKey()
			testType := genTestType()
			a := genRandString()
			b := genRandString()
			thisCase := fmt.Sprintf("%s %s %s", testType, a, b)
			s += fmt.Sprintf("%s %s\n", apiKey, thisCase)

			// don't create an api key entry in 10% of cases
			if rand.Intn(10) == 1 {
				continue
			}

			_, err = conn.Do("SETEX", apiKey, int(2*time.Hour.Seconds()), true)
			if err != nil {
				log.Println("unable to set api key with expiration ", err.Error())
			}

			_, err = conn.Do("SETEX", fmt.Sprintf("%s_%d", batchName, j), int(2*time.Hour.Seconds()), solution(thisCase))
			if err != nil {
				log.Printf("unable to set solution with expiration (%s) - %s", thisCase, err.Error())
			}
		}
	}

	return s
}

func genBatchName() string { return uuid.New() }

func genAPIKey() string { return uuid.New() }

func genTestType() string {
	rand.Seed(time.Now().UnixNano())
	r := rand.Intn(4)
	switch r {
	case 0:
		return "union"
	case 1:
		return "intersection"
	case 2:
		return "concat_sort"
	case 3:
		return "mangle"
	}

	log.Println("error - incorrect test type generated")
	return ""
}

func genRandString() string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZñéíᅒ")

	b := make([]rune, RandStringLen)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)

}

func solution(thisCase string) string {
	return "asdf"
}
