package main

import (
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/docopt/docopt-go"
	"github.com/garyburd/redigo/redis"
	log "github.com/ngaut/logging"
)

var (
	keys    = 10000
	address = "127.0.0.1:19000"
	stop    = true
	bench   = false
)

var usage = `usage: cachetest [--address <127.0.0.1:19000>] [--keys <number_of_keys>] [--nostop <0 or 1>] [--bench <0 or 1>]

options:
   --address set proxy address
   --keys set number of test keys   
   --nostop send commands recursively
   --bench test benchmark
`

var banner string = `
 ____   ____ _   _  ____    _    ____  
|  _ \ / ___| \ | |/ ___|  / \  |  _ \ 
| |_) | |   |  \| | |     / _ \ | | | |
|  __/| |___| |\  | |___ / ___ \| |_| |
|_|    \____|_| \_|\____/_/   \_\____/ 
`

func handleSetLogLevel(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	level := r.Form.Get("level")
	log.SetLevelByString(level)
	log.Info("set log level to", level)
}

func testRedisConn() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	fmt.Println(fmt.Sprintf("-----------SEND %d COMMANDS", keys))
	c, err := redis.Dial("tcp", address)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()
	// send & receive
	val := strings.Repeat("v", 128)
	if err != nil {
		log.Fatalf(err.Error())
	}
	go func() {
		for i := 0; i < keys; i++ {
			c.Send("SET", fmt.Sprintf("TEST-%d", i), val)
		}
	}()
	go func() {
		for i := 0; i < keys; i++ {
			c.Receive()
		}
	}()
	time.Sleep(time.Second * 5)
	fmt.Print(banner)
	fmt.Println("-----------TEST COMPLETE----------")
}

func testRedisConnNOSTOP() {
	fmt.Println("-----------SEND COMMANDS NOSTOP-------------")
	counter := 0
	runtime.GOMAXPROCS(runtime.NumCPU())
	c, err := redis.Dial("tcp", address)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()
	// send & receive
	val := strings.Repeat("v", 128)
	go func() {
		for {
			for i := 0; i < keys; i++ {
				c.Send("SET", fmt.Sprintf("TEST-%d", counter+i), val)
			}
			counter += keys
			fmt.Sprintf("---------HAVE SENT %d Commands--------------", counter)
		}
	}()
	go func() {
		for {
			for i := 0; i < keys; i++ {
				c.Receive()
			}
		}
	}()
	time.Sleep(time.Second * 50)
	fmt.Print(banner)
	fmt.Println("-----------TEST COMPLETE----------")
}

func testBenchmark() {
	fmt.Println("-----------TEST BENCHMARK-------------")
	runtime.GOMAXPROCS(runtime.NumCPU())
	c, err := redis.Dial("tcp", address)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()
	// send & receive
	val := "foo"
	c.Send("SET", val, val)
	go func() {
		for {
			for i := 0; i < keys; i++ {
				c.Send("GET", val)
			}
		}
	}()
	time.Sleep(time.Second * 60)
	fmt.Print(banner)
	fmt.Println("-----------TEST COMPLETE----------")
}

func main() {
	args, err := docopt.Parse(usage, nil, true, "codis proxy v0.1", true)
	if err != nil {
		log.Error(err)
	}
	if args["--address"] != nil {
		address = args["--address"].(string)
	}

	// set cpu
	if args["--keys"] != nil {
		keys, err = strconv.Atoi(args["--keys"].(string))
		if err != nil {
			log.Fatal(err)
		}
	}
	if args["--nostop"] != nil {
		nostop, err := strconv.Atoi(args["--nostop"].(string))
		if err != nil {
			log.Fatal(err)
		}
		if nostop == 1 {
			stop = false
		}
	}
	if args["--bench"] != nil {
		b, err := strconv.Atoi(args["--bench"].(string))
		if err != nil {
			log.Fatal(err)
		}
		if b == 1 {
			bench = true
		}
	}
	fmt.Print(banner)
	log.SetLevelByString("info")
	if bench {
		testBenchmark()
	} else {
		if stop {
			testRedisConn()
		} else {
			testRedisConnNOSTOP()
		}
	}
}
