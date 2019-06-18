# Fanatics Code Challenge

## Instructions

This app is built using docker-compose and therefore you must have docker installed.

Clone the repo and run application using:

`docker-compose up`

The app should start along with a redis instance in docker containers, you will need to make sure port 8080 on localhost is open or change the ports config in `docker-compose.yaml`

_Note: You may need to edit the docker-compose.yaml, specifically the lines:_

```
volumes:
 - .:/go/src/fanatics
working_dir: /go/src/fanatics
```       

_To match where you have cloned the repo, if your $GOPATH is non default or you cloned outside of it - I'm a total newb to golang and was just following a tutorial on setting up a go app with redis using docker compose so not sure about this, sorry!_  
       
The host domain for the server is set to `example.com` in code and should be available immediately by running:

`curl 'http://localhost:8080/cert/example.com'`

New domains may be requested using:

`curl 'http://localhost:8080/cert/bleh.com'`

_Note: Initial domain requests return an empty 202 accepted response, subsequent requests will return the certificate once it's been generated_

## System abstract

I implemented a redis cache to solve most of the requirements for this project.

The 10 minute expiration is set when certs are added so that they auto expire, and a listening pub/sb subscription to redis catches the host certificate expiring and will re-add its domain to the job queue to regenerate on expiration

To handle the issue of the 10 second delay causing concurrency problems a queue and worker solution was implemented, I took the liberty of returning a 202 if the domain is not already cached in redis, a second request will yield the desired result once the certificate has been generated.

This could be improved by implenting a post endpoint with a callback url to allow the client to make a web hook request to receive to generated cert once its available, but I tried to keep it simple for the exercise

Please reach out to me by email or phone if there are any issues or questions.

Thanks!

