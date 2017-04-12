package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
  "encoding/gob"
  "strconv"

	"rallets-cli/core"
)

var config struct {
	Verbose    bool
	UDPTimeout time.Duration
}

var flags struct {
	Version bool
}

type appconfig struct {
  LastSessionId string
}


func logf(f string, v ...interface{}) {
	if config.Verbose {
		log.Printf(f, v...)
	}
}

func fatal(v ...interface{}) {
  fmt.Printf("Error: %s\n", v...)
  os.Exit(1)
}

func SaveAppConfig(c appconfig) error {
  dir := os.Getenv("HOME") + "/.config/rallets-cli"
  path := dir + "/config"
  os.MkdirAll(dir, 0775)
  file, err := os.Create(path)
  if err == nil {
    encoder := gob.NewEncoder(file)
    encoder.Encode(c)
  }
  file.Close()
  return err
}

func ReadAppConfig(c *appconfig) error {
  dir := os.Getenv("HOME") + "/.config/rallets-cli"
  path := dir + "/config"
  file, err := os.Open(path)
  if err == nil {
    decoder := gob.NewDecoder(file)
    err = decoder.Decode(c)
  }
  file.Close()
  return err
}

func login(username string, password string) {
  lgResp, err := RalletsLogin(username, password)
  if err != nil {
    fatal(err)
  }
  var appconfig appconfig
  appconfig.LastSessionId = lgResp.Session_id
  SaveAppConfig(appconfig)
}

func heartbeat() *HeartBeatResponse {
  var appconfig appconfig
  e := ReadAppConfig(&appconfig)
  if e != nil {
    fatal("Please login first")
  }
  hbResp, err := RalletsHeartBeat(appconfig.LastSessionId)
  if err != nil {
    fatal(err)
  }
  return hbResp
}

func usage() {
  fmt.Println(`
Usage: rallets-cli COMMAND

Commands:
login     Login to rallets
ls        List available servers
connect   connect to a server by id

Example 1:
rallets-cli login youruser yourpassword
>> 7b0f1ce4-xxxx-xxxx SERVER1
>> ebf14aed-xxxx-xxxx SERVER2
>> a87f57b6-xxxx-xxxx SERVER3
nohup rallets-cli connect 7b0 >/tmp/rallets.log 2>&1 &

Example 2:
If you have logined before, you can directly list servers by
rallets-cli ls

Closing rallets:
killall rallets-cli

Note:
When connecting to a server, server id can be in short form
If server id is '7b0f1ce4-xxxx-xxxx'
You can issue the 'connect' by 'connect 7b' or 'connect 7b0'
`)
}

func main() {
  flag.Usage = func() {
    usage()
  }
	flag.BoolVar(&flags.Version, "v", false, "Show rallets-cli version")
	flag.BoolVar(&config.Verbose, "verbose", false, "Verbose mode")
	flag.DurationVar(&config.UDPTimeout, "udptimeout", 5*time.Minute, "UDP tunnel timeout")
	flag.Parse()

  cmdargs := flag.Args()

  if (flags.Version) {
    fmt.Println(VERSION)
    return
  }

  if len(cmdargs) <= 0 {
    flag.Usage()
    return
  }

  switch cmdargs[0] {

  case "login":
    username := cmdargs[1]
    password := cmdargs[2]
    login(username, password)
    fallthrough

  case "ls":
    ss := heartbeat().Self.Ssconfigs
    for i := 0; i < len(ss); i++ {
      fmt.Println(ss[i].Id[0:8] + " " + ss[i].Remarks)
    }
    return

  case "connect":
    serverShort := cmdargs[1]
    hb := heartbeat()
    ss := hb.Self.Ssconfigs
    found := false
    for i := 0; i < len(ss); i++ {
      if strings.HasPrefix(ss[i].Id, serverShort) {
        found = true
        thess := ss[i]
        var err error
        server := thess.Server + ":" + thess.Server_port
        method := strings.ToUpper(thess.Method)
        password := thess.Password
        ciph, err := core.PickCipher(method, []byte(""), password)
        if err != nil {
          fatal(err)
        }
        go socksLocal(":" + strconv.Itoa(thess.Port), server, ciph)
        startHeartbeat(&thess)
        break
      }
    }
    if !found {
      fatal("Server ID [" + serverShort + "] not found")
    }

  default:
    fatal("Unknown command " + cmdargs[0])
  }

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
}

func startHeartbeat(currSS *Ssconfig) {
  ticker := time.NewTicker(time.Second * HEARTBEAT_INTERVAL)
  go func() {
    newVersionAsked := false
    for range ticker.C {
      hb := heartbeat()
      if !hb.Ok {
        fatal(hb.Message)
      }
      nt := hb.SystemNotification
      if VERSION < nt.Version && !newVersionAsked {
        fmt.Println("A new version is available - " + nt.Version)
        fmt.Println("Download at: " + nt.Download_link)
        newVersionAsked = true
      }
      if nt.Show {
        fmt.Println(nt.Message)
        if (strings.HasPrefix(nt.Link, "http")) {
          fmt.Println(nt.Link)
        }
      }
      // Check if current connected server is still there
      ss := hb.Self.Ssconfigs
      found := false
      for i := 0; i < len(ss); i++ {
        if profileIdentical(ss[i], *currSS) {
          found = true
          break
        }
      }
      if !found && len(ss) > 0 {
        // Connect to a random server
        *currSS = ss[0]
        // log.Println("Server updated")
      }
    }
  }()
}

func profileIdentical(a Ssconfig, b Ssconfig) bool {
  return a.Port == b.Port &&
         a.Method == b.Method &&
         a.Server == b.Server &&
         a.Password == b.Password &&
         a.Server_port == b.Server_port
}
