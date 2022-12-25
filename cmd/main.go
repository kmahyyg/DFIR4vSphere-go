package main

import (
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/kmahyyg/DFIR4vSphere-go/pkg/common"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"os/signal"
	"sync"
	"time"
)

func init() {
	// create log file
	var err error
	common.LogFileFD, err = os.Create("working.log")
	if err != nil {
		panic(err)
	}
	logMWriter := io.MultiWriter(common.LogFileFD, os.Stderr)
	log.SetOutput(logMWriter)
	// set json format
	log.SetFormatter(&log.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
	})
	// check if debug
	if os.Getenv("RunEnv") == "DEBUG" {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
}

func main() {
	// cleanup via defer
	defer func() {
		if common.LogFileFD != nil {
			common.LogFileFD.Sync()
			common.LogFileFD.Close()
		}
	}()
	// log software version for debugging
	log.Infoln("Software Version: " + common.VersionStr)
	fmt.Println("[+] DFIR4vSphere-go - " + common.VersionStr)
	// survey questions
	qslist := []*survey.Question{
		{
			Name: "http_proxy", // ask proxy
			Prompt: &survey.Input{
				Message: "HTTP Proxy URL? (http://HOST:PORT) (If not, press enter) ",
				Help:    "Example: \"http://127.0.0.1:3128\".",
			},
		},
		{
			Name: "vsphere_hostport",
			Prompt: &survey.Input{
				Message: "vSphere URL? (https://HOST:PORT, do not ignore port, default 443) ",
				Help:    "Example: \"https://192.168.56.128:443\", vCenter URL or ESXi URL here ",
			},
			Validate: survey.Required,
		},
		{
			Name: "vsphere_user",
			Prompt: &survey.Input{
				Message: "Administrator Username? ",
				Help: "By default, vCenter use: administrator@vsphere.local, ESXi use: root; " +
					"if you are not using password-based authentication, we are not supported currently.",
			},
			Validate: survey.Required,
		},
		{
			Name:     "vsphere_pass",
			Prompt:   &survey.Password{Message: "Administrator Password? "},
			Validate: survey.Required,
		},
		{
			Name: "standalone_esxi",
			Prompt: &survey.Confirm{
				Message: "Is input host a standalone ESXi Host? ",
				Default: false,
				Help:    "If your URL is vCenter, choose false. Else, choose true.",
			},
		},
	}
	// ask and get answer
	err := survey.Ask(qslist, common.UserAnswer)
	if err != nil {
		panic(err)
	}
	log.Infoln("User Answer: " + common.UserAnswer.String())
	log.Debugln("User Password: " + common.UserAnswer.Password)
	// user input done

	// start build connection
	// if not working, detect error and warn user then exit

	// ask user cmd
	// but build output file first

	// defer to cleanup

	// handle signal
	var sigChan = make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, os.Kill)
	// tasks might be async, especailly download support bundle, must wait
	// max timeout 30mins
	var wgBackground = &sync.WaitGroup{}
	// further cmd
	go func() {
		wgBackground.Add(1)
		defer wgBackground.Done()
		// query input
		for {
			var nextCmd string
			err := survey.AskOne(&survey.Input{
				Message: "[>_] $ ",
				Help:    "Supported commands: [support_bundle output=filepath] [test_connection] [basic_info] [vi_events light_mode=bool] [exit] [full_help]",
			}, &nextCmd, survey.WithValidator(survey.Required))
			if err != nil {
				log.Fatalln(err)
			}
			switch nextCmd {
			case "exit":
				sigChan <- os.Interrupt
				close(sigChan)
				return
			case "full_help":

			}
		}
	}()
	<-sigChan
	fmt.Println("Exit signal received. Waiting for background tasks, max timeout 30 mins. " +
		"Press Ctrl-C again to force exit, but you may experience unexpected data loss.")
	wgBackground.Wait()
	fmt.Println("Background tasks done. Cleaning up... Exit after 3 seconds.")
	time.Sleep(3 * time.Second)
}
