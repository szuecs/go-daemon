package main

import (
	"fmt"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/google/gops/agent"
	"github.com/szuecs/go-daemon/client"
	gomonitor "github.com/zalando/gin-gomonitor"
	"github.com/zalando/gin-gomonitor/aspects"
	"gopkg.in/alecthomas/kingpin.v2"
	"gopkg.in/mcuadros/go-monitor.v1/aspects"
	resty "gopkg.in/resty.v0"
)

//Buildstamp and Githash are used to set information at build time regarding
//the version of the build.
//Buildstamp is used for storing the timestamp of the build
var Buildstamp = "Not set"

//Githash is used for storing the commit hash of the build
var Githash = "Not set"

// Version is used to store the tagged version of the build
var Version = "Not set"

var cli *client.Client

const (
	productionAPI  = "https://prodAPI"
	integrationAPI = "https://integrationAPI"
)

func main() {
	var (
		env              = kingpin.Flag("env", "Environment, API to call <prod|*>").Envar("ENV").Default("integration").String()
		numGoroutinesPtr = kingpin.Flag("goroutine", "Number of goroutines to spawn.").Envar("GOROUTINES").Int()
		token            = kingpin.Flag("token", "Token.").Default(os.ExpandEnv("$TOKEN")).String()
		quiet            = kingpin.Flag("quiet", "enable quiet mode").Default("false").Bool()
		debug            = kingpin.Flag("debug", "enable debug mode").Default("false").Bool()
		version          = kingpin.Flag("version", "show version").Default("false").Bool()
	)

	kingpin.Parse()

	if *version {
		fmt.Printf(`%s Version: %s
================================
    Buildtime: %s
    GitHash: %s
`, path.Base(os.Args[0]), Version, Buildstamp, Githash)
		os.Exit(0)
	}

	// monitoring
	genericAspect := ginmon.NewGenericChannelAspect("generic")
	genericAspect.StartTimer(1 * time.Minute)
	genericCH := genericAspect.SetupGenericChannelAspect()

	router := gin.New()
	// curl http://localhost:9000/RequestTime
	router.Use(gomonitor.Metrics(9000, []aspects.Aspect{genericAspect}))

	numGoroutines := *numGoroutinesPtr

	if *debug {
		log.SetLevel(log.DebugLevel)
		log.Debugln("Debug enabled")
	} else if *quiet {
		log.SetLevel(log.WarnLevel)
	}
	if err := agent.Listen(nil); err != nil {
		log.Errorf("Gops agent could not start %v", err)
	}

	switch *env {
	case "prod":
		cli = client.NewClient(productionAPI, *token, *debug)
		log.Debug("Use productionAPI")
	default:
		cli = client.NewClient(integrationAPI, *token, *debug)
		log.Debug("Use integrationAPI")
	}

	// configure http rest client, Resty
	resty.SetRedirectPolicy(resty.FlexibleRedirectPolicy(10))

	if numGoroutines == 0 {
		// default: only a single goroutine
		numGoroutines = 1
	}
	shutdown := make(chan bool, 1)
	stopped := make(chan bool, 1)

	log.Printf("spawn %d workers", numGoroutines)
	for i := 1; i <= numGoroutines; i++ {
		go Mainloop(&shutdown, &stopped, genericCH)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	// Block until a signal is received.
	s := <-sigs
	log.Println("signal shutdown, caused by", s)
	for i := 1; i <= numGoroutines; i++ {
		shutdown <- true
	}
	for i := 1; i <= numGoroutines; i++ {
		<-stopped
		log.Printf("stopped worker %d", i)
	}
}

// Mainloop is the implementation of one worker. You can signal a
// worker to shutdown, if you send it a message to shutdown channel
// and it will send you back a message on the stopped channel.
func Mainloop(shutdown, stopped *chan bool, genericCH chan ginmon.DataChannel) {
	for {
		now := time.Now()
		select {
		case <-*shutdown:
			*stopped <- true
		default:
		}

		// TODO: do your hard work
		time.Sleep(3 * time.Second)

		genericCH <- ginmon.DataChannel{Name: "Main loop", Value: float64(time.Now().Sub(now))}
	}
}
