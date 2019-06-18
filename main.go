package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"os"
	redis "gopkg.in/redis.v4"
	"net/http"
	"time"
)

//10 minute TTL for generated certificates
const certExpiration time.Duration = time.Second * 60 * 10

//host name for the local server, used for local cert gen
const hostDomain string = "example.com"

//redis pubsub subscription string to listen for host cert expiration regeneration
const hostCertSub string = "__keyspace@*__:" + hostDomain

//dummy filler for mocking certificate generation
const certPrefix string = "foo"

//time for workers to sleep between cert gen
const sleepTime time.Duration = time.Second * 10

//number of workers
const workerCount int = 1


//check redis for existing cert and return
func getCertificate(client *redis.Client, domainName string) string {
	val, err := client.Get(domainName).Result()
	if err != nil {
		return ""
	}
	return val
}

//set cert in redis
func putCertificate(client *redis.Client, domain string, certificate string) {
	//make call to redis to set certificate
	err := client.Set(domain, certificate, certExpiration).Err()
	if err != nil {
		fmt.Println("Error generating certifcate for: '" + domain + "'" ,err)
	}
}

//struct to hold incoming cert jobs in queue
type Job struct {
	domain string       //domain name for cert
}

//generate cert, add to redis
func generateCertificate(job Job, client *redis.Client) {
	//generate certificate by prepending prefix to domain name
	certificate := certPrefix + "-" + job.domain

	putCertificate(client, job.domain, certificate)
}

//worker func, reads from job channel and generates cert, then sleeps
func worker(jobChan <-chan Job, cancelChan <-chan struct{}, client *redis.Client) {
	for {
		select {
		case <-cancelChan:
			return

		case job := <-jobChan:
			generateCertificate(job, client)
		}
		//sleep worker for specified time
		time.Sleep(sleepTime)
	}
}

func main() {
	//Setup client/connection to redis instance
	redisClient := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_URL"),
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	// create a cancel channel
	cancelChan := make(chan struct{})

	//create a job channel
	jobChan := make(chan Job)

	//create worker pool
	for i:=0; i<workerCount; i++ {
		go worker(jobChan, cancelChan, redisClient)
	}

	//generate host certificate
	jobChan <- Job{hostDomain}

	//subscribe to redis keyspace events and look for expiration of
	// the main certificate, on expire generate new one
	go func(){
		psNewMessage, _ := redisClient.PSubscribe(hostCertSub)
		for {
			msg, err := psNewMessage.ReceiveMessage()

			if err != nil {
				fmt.Println("Error inside redis subscription: ", err)
			}

			fmt.Println("got msg: '" + msg.String() + "'")

			fmt.Println("Received host certificate expiration message, regenerating")
			jobChan <- Job{hostDomain}
		}
	}()

	//setup mux router for the GET/POST /cert/:domain endpoint
	router := mux.NewRouter()
	router.HandleFunc("/cert/{domain}", func(w http.ResponseWriter, request *http.Request) {
		vars := mux.Vars(request)
		domain := vars["domain"]

		cert := getCertificate(redisClient, domain)

		if cert != "" {
			//write cert to body and return as status OK
			_, err := fmt.Fprintf(w, cert)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
		} else {
			//enqueue job and return 202 Accepted
			jobChan <- Job{domain}
			w.WriteHeader(http.StatusAccepted)
		}
	}).Methods("GET")
	http.Handle("/", router)

	//start http server
	http.ListenAndServe(":8080", nil)

	//cancel running workers
	close(cancelChan)
}
