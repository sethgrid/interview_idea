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
	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

const (
	// function types
	Union        = "union"
	Intersection = "intersection"
	UnionSort    = "unionsort"
	Mangle       = "mangle"
)

// Globals - yeah, I know

// populated with flags:
var Port int
var RandStringLen int
var MaxBatchSize int
var DefaultBatchCount int
var RedisAddr string
var ShowSolutions bool

// populated internally
var APIKeyPool []string

func init() {
	flag.IntVar(&Port, "port", 9090, "set the port as an integer")
	flag.IntVar(&RandStringLen, "string-length", 15, "set the random string length")
	flag.IntVar(&MaxBatchSize, "max-in-batch", 15, "max number of cases in each batch")
	flag.IntVar(&DefaultBatchCount, "batches", 10, "number of batches to generate by default (overwrite with query params)")
	flag.StringVar(&RedisAddr, "redis", ":6379", "set host and port for redis")
	flag.BoolVar(&ShowSolutions, "show-solutions", false, "set to see solutions")
	setAPIKeyPool()
}

func main() {
	flag.Parse()
	log.Printf("starting on localhost:%d. URL has instructions.", Port)

	r := mux.NewRouter()
	r.HandleFunc("/", rootHandler)
	r.HandleFunc("/generate", generateHandler)
	r.HandleFunc("/validate/apikey/{key}", validateAPIKeyHandler)
	r.HandleFunc("/validate/batch/{batch}", validateBatchHandler)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", Port), r))
}

// provide sinsible information for a candidate to use this service
func rootHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(fmt.Sprintf(`
<html>
<head><title>Interview</title></head>
<body>

<h1>Interview</h1>
Curl or go to <a href="http://localhost:%d/generate?count=10">http://localhost:%d/generate?count=10</a>
<br>
Feel free to adjust count.
<br><br>
The data you see is in the form:
<br>
<pre>
B C
APIKEY FUNCTION STRING_A STRING_B
APIKEY FUNCTION STRING_A STRING_B
</pre>
Where B is the name of the batch. C is the number of elements/work-items in this batch. Each work-item will have an api key that must be validated at /validate/apikey/:APIKEY.
Non valid api keys should not be allowed to request work to be processed. These entries should report "invalid" as the solution.
<br>
<br>
The function can be one of the following:
<br>
%s: get the characters that appear in both strings
<br>
%s: concat the two strings together, remove duplicates, and preserve order
<br>
%s: concat the strings and sort them (assuming American English as the guide for letter priority)
<br>
%s: take the even indexed letters from the first string and the odd indexed letters from the second string
<br>
<br>
Each batch can be verified for correctness by submitting to /validate/batch/:B with a post body where each line represents the solution to the corresponding work request.
<br>
<pre>
Example (note: there are not duplicate characters in solutions):
2
Foo
some-key intersection apples planes
some-other-key union apples planes

Solution:
curl -X POST localhost:%d/validate/batch/Foo -d 'pes
aplesn
'
</pre>
</body>
</html>
`, Port, Port, Intersection, Union, UnionSort, Mangle, Port)))
}

// create an input data set that the candidate will work against
func generateHandler(w http.ResponseWriter, r *http.Request) {
	c := r.URL.Query().Get("count")
	if c == "" {
		c = strconv.Itoa(DefaultBatchCount)
	}

	count, err := strconv.Atoi(c)
	if err != nil {
		handleErr(w, http.StatusBadRequest, "count param should be a number")
		return
	}

	w.Write([]byte(genInput(count)))
}

// validate a batch of data submitted by the candidate
func validateBatchHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	batch, ok := vars["batch"]
	if !ok {
		handleErr(w, http.StatusBadRequest, "must provide batch uuid as URI segment")
		return
	}

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

	/*
	   Expected body format, n lines, each representing an ordered response based on batch
	   ex:

	   dasdfSDFd
	   sdfHDFas
	   invalid
	   DGHDksdfhkdL

	*/

	// validate each line individually
	submissionValid := true
	lines := strings.Split(string(body), "\n")
	for i, line := range lines {
		if line == "" {
			continue
		}
		key := fmt.Sprintf("%s_%d", batch, i)

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
			submissionValid = false
			w.Write([]byte(fmt.Sprintf("invalid submission %s_%d got %s, want %s\n", batch, i, line, data)))
		}
	}

	if submissionValid {
		w.Write([]byte("ok\n"))
	}
}

// check if the apikey is valid (lives in redis)
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

func genInput(count int) string {
	conn, err := redis.Dial("tcp", RedisAddr)
	if err != nil {
		log.Println("unable to connect to redis when generating input ", err.Error())
		return "<< error >>"
	}

	var s string
	for i := 1; i <= count; i++ {
		batchName := genBatchName()
		batchSize := rand.Intn(MaxBatchSize)
		s += fmt.Sprintf("%s %d\n", batchName, batchSize)

		for j := 0; j <= batchSize; j++ {
			apiKey := genAPIKey()
			testType := genTestType()
			a := genRandString()
			b := genRandString()
			thisCase := fmt.Sprintf("%s %s %s %s", apiKey, testType, a, b)
			answer := solution(thisCase)
			if ShowSolutions {
				s += fmt.Sprintf("%s # %s\n", thisCase, answer)
			} else {
				s += fmt.Sprintf("%s\n", thisCase)
			}

			_, err = conn.Do("SETEX", fmt.Sprintf("%s_%d", batchName, j), int(2*time.Hour.Seconds()), answer)
			if err != nil {
				log.Printf("unable to set solution with expiration (%s) - %s", thisCase, err.Error())
			}
		}
	}

	return s
}

func genBatchName() string { return uuid.New() }

func setAPIKeyPool() {
	APIKeyPool = []string{
		"012a782f-9c51-4a18-b6b9-77295bea63cc",
		"599549f7-6a01-469c-8be1-aa32d2e1bd68",
		"04e40469-e5b7-4bce-a742-878fe43e0917",
		"4255c2d8-10db-4d45-a7fb-af22b214b4be",
		"5f21c480-fd4c-43ab-bb74-918ff834eff2",
		"004fbf26-ac58-4534-9c03-fc61b508b046",
		"faea0096-9990-4dfc-b179-2f21b3392670",
		"045d383e-b9c5-4024-9c9b-c4e0cb04cfe2",
		"c26ad75e-1fd3-4aab-9797-6ec5337e9cf9",
		"d1a0ae11-1052-40a7-a8cd-83192c8f5444",
		"7411f5a0-d476-498a-b329-c3321b3d3d66",
		"1030dec9-e8d6-4042-af52-2230313d5204",
		"44c425b9-7109-495b-a897-b57fa87da09e",
		"1c49b816-b488-4caa-8752-f4780dde9730",
		"0ca20441-6c3d-4fb1-aafc-f7949e71105a",
		"ec3a40c5-5279-42ef-bb9f-2c77752ac072",
		"d5e64ebe-645f-4be7-b20a-62f6a5bb6415",
		"5d68a3ba-c05e-4114-b6cf-6427c6ea38d3",
		"d1bec9a7-fd67-43a3-a2e9-7ae085063b2f",
		"097d38bf-27ae-406d-914b-44bd95ea6d05",
		"9f1717b6-8fa4-407e-8850-94ee5d5a91fc",
		"d4c38cdb-77b4-468b-81dd-ccc83171067d",
		"f1a658ef-73bd-4e4d-8922-9fefc77eaff0",
		"aca0c41d-5a97-46e6-8a11-3872dcd3f60a",
		"60c5be58-37db-4ace-90b5-b54e6c44527f",
		"03ab7b0f-abdc-4991-9b8b-fa57b2e35090",
		// "ede2cf62-27a6-4122-afde-358a317935b1",
		// "f410747f-2e81-45b3-9633-733c4be61cea",
		// "a7ff9400-e609-4908-8c4d-031f418a57dc",
		// "56a0ad3c-bbfd-4f7f-b2cb-0c928d5ecc51",
		// "c38de002-15b2-4f97-a4f1-d521e9fc46a8",
		// "98958321-73de-4e85-961c-3dbbe37b78a7",
		// "8e00e59f-ee5b-45ab-a975-91f0f3d51cb7",
		// "f0f36997-da3e-47cf-9f55-bb9d56b8ed78",
		// "9e5ea1e4-ef58-4067-a5e8-c5f240044450",
		// "3a3d6269-b780-4de3-aded-33be5b45c7cc",
		// "92fc7307-823e-47cc-8302-3957ac769b8e",
		// "54df4ec7-21b5-4567-8311-24644ab54fa3",
		// "157cd7f0-f3a1-4b0e-9f13-8bf2a7fdce69",
		// "20539e8e-bdf4-4129-98b5-8328e8322c92",
		// "cd8858e2-e42b-4975-aadb-58f08c8c2534",
		// "3d84a97f-9a16-435d-b149-3b982d758201",
		// "970cca57-c69a-42ce-91a6-06e60dd6a3d8",
		// "ccb75279-6af9-4cf1-9298-e66cb6394c1e",
		// "99f2ae28-d346-43e9-825d-62e00c2128c5",
		// "af997a41-2b29-4f12-a9a2-6cac5f5bd450",
		// "0e3cb9dd-a158-47e4-9e1b-4cb26ff3d46b",
		// "437c13a0-4d1d-4664-b8a6-992ef4f12b3a",
		// "571106c8-953a-4773-8ec1-02dfa292e314",
		// "847d2125-8607-446e-996b-d2cf6ff9ac9a",
		// "1a5bd82e-5a04-421d-aca9-7294df428756",
		// "020ac153-67dc-467f-837e-ad720eb6628b",
		// "304bab5d-11fd-41b3-be0e-494515ae4d54",
		// "f772c251-a481-4ccc-bb73-6f2230b94763",
		// "12b34ba7-cae0-4696-b353-bba71b90a1a2",
		// "dc65992e-0fac-48ba-bd9c-2929db88ba10",
		// "432bcd93-66a2-4e77-bdca-9327aabb48b2",
		// "5699a2c7-b6c8-48ea-9cab-696b8e05f282",
		// "8698d6b4-e0c9-406e-9080-745ff7fb811e",
		// "cdd70e83-82cc-41f2-9927-c6d892e6200f",
		// "69f9926d-5352-4115-b87a-f273f00fe8e7",
		// "9dc7257b-c1ee-4d95-99f4-164a4d52aed0",
		// "9de92095-e32e-4039-b165-e3c4e203a553",
		// "4d487bde-d8ff-4f6c-97ce-f33d3163498f",
		// "e0ba455a-361c-49bb-a8c5-3fa92f242227",
		// "b8d69fed-cb8e-4c93-a7ea-f3a827776e71",
		// "31563061-82da-4bc6-b244-a577a6a4f6c4",
		// "0b667e50-c318-4ea4-abb5-2c0d615c5c1c",
		// "4f4f1c44-586a-4d38-8080-a18082d08bd9",
	}

	conn, err := redis.Dial("tcp", RedisAddr)
	if err != nil {
		log.Fatalf("unable to dial redis - %s", err)
	}
	for _, apiKey := range APIKeyPool {
		_, err = conn.Do("SETEX", apiKey, int(2*time.Hour.Seconds()), true)
		if err != nil {
			log.Println("unable to set api key with expiration ", err.Error())
		}
	}
}

func genAPIKey() string {
	// In 10% of cases, generate a new uuid.
	// This will result in an apikey that will not validate
	// as it was not set in redis
	if rand.Intn(10) == 1 {
		return uuid.New()
	}
	return APIKeyPool[rand.Intn(len(APIKeyPool))]
}

func genTestType() string {
	rand.Seed(time.Now().UnixNano())
	r := rand.Intn(4)
	switch r {
	case 0:
		return Union
	case 1:
		return Intersection
	case 2:
		return UnionSort
	case 3:
		return Mangle
	}

	log.Println("error - incorrect test type generated")
	return ""
}

// solution generates the solution for the given case
func solution(thisCase string) string {
	// check that the api key is in redis, if not, set response to "invalid"
	parts := strings.Split(thisCase, " ")
	if len(parts) != 4 {
		log.Printf("error - solution(%s): not four parts", thisCase)
		return "<< error parsing data, not four parts: " + thisCase + " >>"
	}

	apiKey := parts[0]
	function := parts[1]
	a := parts[2]
	b := parts[3]

	conn, err := redis.Dial("tcp", RedisAddr)

	ok, err := redis.Bool(conn.Do("GET", apiKey))
	if err != nil && err != redis.ErrNil {
		log.Println("unable to get api key for solution generation ", err.Error())
		return "<< internal redis error getting data >>"
	}

	if !ok {
		return "invalid"
	}

	switch function {
	case Union:
		return union(a, b)
	case Intersection:
		return intersection(a, b)
	case UnionSort:
		return unionSort(a, b)
	case Mangle:
		return mangle(a, b)
	}

	log.Println("error - unexpected function ", function)
	return "<< unexpected function: " + function + " >>"
}

func deduplicate(s string) string {
	var deduplicated string
	runes := make(map[rune]bool)
	for _, r := range s {
		if _, ok := runes[r]; !ok {
			runes[r] = true
			deduplicated += string(r)
		}
	}
	return deduplicated
}

func intersection(a, b string) string {
	var s string
	for _, runeA := range a {
		for _, runeB := range b {
			if runeA == runeB {
				s += string(runeA)
			}
		}
	}
	return deduplicate(s)
}

func union(a, b string) string {
	return deduplicate(a + b)
}

func unionSort(a, b string) string {
	j := make([]string, 0)
	for _, aChars := range a {
		j = append(j, string(aChars))
	}
	for _, bChars := range b {
		j = append(j, string(bChars))
	}

	collate.New(language.AmericanEnglish, collate.OptionsFromTag(language.AmericanEnglish)).SortStrings(j)
	return deduplicate(strings.Join(j, ""))
}

func mangle(a, b string) string {
	// grab even indexed chars from a, odd from b
	var s string

	for i, runeA := range a {
		for j, runeB := range b {
			if j != i {
				continue
			}

			if i%2 == 0 {
				s += string(runeA)
			} else {
				s += string(runeB)
			}
		}

	}
	return deduplicate(s)
}

func genRandString() string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZñéíᅒ")

	b := make([]rune, RandStringLen)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)

}

// helper function for making error responses look the same
func handleErr(w http.ResponseWriter, statusCode int, msg string) {
	w.WriteHeader(statusCode)
	w.Write([]byte(msg + "\n"))
}
