package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"code.google.com/p/go-uuid/uuid"
	"github.com/gorilla/mux"
)

var Port int
var RandStringLen int
var MaxBatchSize int

func init() {
	flag.IntVar(&Port, "port", 9090, "set the port as an integer")
	flag.IntVar(&RandStringLen, "string-length", 15, "set the random string length")
	flag.IntVar(&MaxBatchSize, "max-in-batch", 15, "max number of cases in each batch")
}

func main() {
	flag.Parse()
	r := mux.NewRouter()

	log.Printf("starting on 0.0.0.0:%d", Port)

	r.HandleFunc("/", rootHandler)
	r.HandleFunc("/generate", generateHandler)

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
	w.Write([]byte("use http://localhost:%d/generate?count=10"))
}

func handleErr(w http.ResponseWriter, statusCode int, msg string) {
	w.WriteHeader(statusCode)
	w.Write([]byte(msg))
}

func genInput(count int) string {
	var s string
	s += fmt.Sprintf("%d\n", count)
	for i := 1; i <= count; i++ {
		batchName := genBatchName()
		s += fmt.Sprintf("batch %s\n", batchName)

		for j := 0; j <= rand.Intn(MaxBatchSize); j++ {
			apiKey := genAPIKey()
			testType := genTestType()
			a := genRandString()
			b := genRandString()

			s += fmt.Sprintf("%s %s %s %s\n", apiKey, testType, a, b)
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
