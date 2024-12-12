package main

import (
	"aes-tracer-service/server"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
)

var port = "8080"

func main() {
	tls, err := strconv.ParseBool(server.GetEnv(server.EnvTLS, "false"))
	if err != nil {
		log.Println("ERROR: ENABLE_HTTPS environment variable must be either 'true' or 'false'.")
	}

	defPort := port
	if tls {
		defPort = "8443"
	}

	port, err := strconv.Atoi(server.GetEnv(server.EnvPORT, defPort))
	if err != nil {
		log.Fatalln(err)
	}

	if port < 1 || port > 65535 {
		log.Fatalln("Server port must be in range 1..65535 (inclusive)")
	}

	s := server.New(
		"aes-tracer",
		os.Getenv(server.EnvHOST),
		port,
		tls,
	)

	s.ConfigureRouter()

	// Handle SIGTERM gracefully
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM)

	go func() {
		<-signals
		s.Ready = false
		log.Println("SIGTERM received. Marked unhealthy and waiting to be killed.")
	}()

	log.Fatalln(s.Start())
}
