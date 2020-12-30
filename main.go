package main

import (
	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type graceListener struct {
	ln net.Listener
}

var gl graceListener
var graceWaitGroup sync.WaitGroup
var maxWaitTime time.Duration = 10 * time.Second

// Start the service and open the web address http://127.0.0.1:666/index (it is set to take 6 seconds to execute)
// close the service immediately, and you will find that it will shut down gracefully
func main() {
	go NewServer()
	GracefulExit()
}

func NewServer() {
	log.Println("start server ... ...")

	ln, err := net.Listen("tcp4", ":666")
	if nil != err {
		log.Println("net.Listen err: ", err.Error())
		os.Exit(-5)
	}
	gl = graceListener{
		ln,
	}

	r := router.New()
	r.GET("/index", func(ctx *fasthttp.RequestCtx) {
		graceWaitGroup.Add(1)
		defer graceWaitGroup.Done()

		_, _ = ctx.Write([]byte(time.Now().Format(" 2006-01-02 15:04:05 ")))
		time.Sleep(time.Second * 6)
		_, _ = ctx.Write([]byte(time.Now().Format(" 2006-01-02 15:04:05 ")))
	})

	fastServ := &fasthttp.Server{
		Concurrency:  100,
		Handler:      r.Handler,
		LogAllErrors: true,
	}
	if err := fastServ.Serve(gl.ln); err != nil {
		log.Println("fasthttp.Serve err: ", err.Error())
		os.Exit(-5)
	}
}

// Receive signal, exit gracefully
func GracefulExit() {
	log.Println("register signal handler")
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT)
	switch <-ch {
	case syscall.SIGTERM, syscall.SIGINT:
		log.Println("get signal SIGTERM, prepare exit!!!")

		err := gl.ln.Close()
		if err != nil {
			log.Println("gl.Close err: ", err.Error())
		}
		// Block until all requests are completed or until the maximum waiting time
		select {
		case <-waitAllRoutineDone():
			log.Printf("waitAllRoutineDone")
		case <-time.After(maxWaitTime):
			log.Printf("force shutdown after %v\n", maxWaitTime)
		}

		log.Println("get signal SIGTERM, success exit!!!")
		break
	}
}

func waitAllRoutineDone() chan struct{} {
	flagChan := make(chan struct{}, 1)
	graceWaitGroup.Wait()
	flagChan <- struct{}{}
	return flagChan
}
