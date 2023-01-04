package main

import (
	"errors"
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/kmahyyg/DFIR4vSphere-go/pkg/common"
	"github.com/kmahyyg/DFIR4vSphere-go/pkg/subcmds"
	"github.com/kmahyyg/DFIR4vSphere-go/pkg/vsphere_api"
	log "github.com/sirupsen/logrus"
	"intra-git.kmahyyg.xyz/kmahyyg/usertelemetry/golang/telemetry"
	"io"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"time"
)

func init() {
	// create log file
	var err error
	common.LogFileFD, err = os.Create("working.log.json")
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
		log.SetReportCaller(true)
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
	// telemetry
	if os.Getenv("IW0ulDL1Ke2OPT0UtFr0MTeLEmETrY") == "" {
		err := telemetry.Collect()
		if err != nil {
			log.Fatalln(err)
		}
	}
	// log software version for debugging
	log.Infoln("Software Version: " + common.VersionStr)
	fmt.Println("[+] DFIR4vSphere-go - " + common.VersionStr)
	// survey questions
	qslist := []*survey.Question{
		{
			Name: "http_proxy", // ask proxy
			Prompt: &survey.Input{
				Message: "HTTP Proxy URL? (http://HOST:PORT) (If not, press enter)",
				Help:    "Example: \"http://127.0.0.1:3128\".",
			},
		},
		{
			Name: "vsphere_hostport",
			Prompt: &survey.Input{
				Message: "vSphere URL? (https://HOST:PORT, default 443)",
				Help:    "Example: \"https://192.168.56.128:443\", vCenter URL or ESXi URL here ",
			},
			Validate: survey.Required,
		},
		{
			Name: "vsphere_user",
			Prompt: &survey.Input{
				Message: "Administrator Username?",
				Help: "By default, vCenter use: administrator@vsphere.local, ESXi use: root; " +
					"if you are not using password-based authentication, we are not supported currently.",
			},
			Validate: survey.Required,
		},
		{
			Name:     "vsphere_pass",
			Prompt:   &survey.Password{Message: "Administrator Password?"},
			Validate: survey.Required,
		},
		{
			Name: "skip_tls_verify",
			Prompt: &survey.Confirm{
				Message: "Skip TLS Certificate Check?",
				Default: false,
				Help:    "Leave it as default unless you know what you are doing.",
			},
		},
	}
	// ask and get answer
	err := survey.Ask(qslist, common.UserAnswer)
	if err != nil {
		panic(err)
	}
	log.Debugln("User Answer: " + common.UserAnswer.String())
	log.Debugln("User Password: " + common.UserAnswer.Password)
	// user input finished
	// start build connection
	// build sdk path
	var proxyURLInstance *url.URL = nil
	if common.UserAnswer.HttpProxyHost != "" {
		var proxyCheckErr error
		proxyURLInstance, proxyCheckErr = url.Parse(common.UserAnswer.HttpProxyHost)
		if proxyURLInstance.Path != "" {
			proxyCheckErr = errors.New("proxy url should not have any path and querystring")
		}
		if proxyCheckErr != nil {
			log.Fatalln("HTTP Proxy Invalid: " + proxyCheckErr.Error())
		}
		log.Infoln("User set to use HTTP Proxy, pre-flight check passed.")
	}
	vcURL, err := url.Parse(common.UserAnswer.HostAddr)
	if vcURL.Scheme != "https" {
		log.Fatalln("vSphere Host should only use HTTPS")
	}
	vcURL.Path = "/sdk"
	finalUserInfoInURL := url.UserPassword(common.UserAnswer.Username, common.UserAnswer.Password)
	vcURL.User = finalUserInfoInURL
	log.Debugln("Final built URL for vSphere: " + vcURL.String())
	// build client
	err = vsphere_api.GlobalClient.Init(vcURL, common.UserAnswer.SkipTLSVerify, proxyURLInstance)
	if err != nil {
		log.Fatalln("Initialize Environment for vSphere Client failed: ", err.Error())
	}
	log.Infoln("vSphere Client Environment Set.")
	err = vsphere_api.GlobalClient.NewClient()
	if err != nil {
		log.Fatalln("Create vSphere Client Instance Failed: " + err.Error())
	}
	log.Infoln("vSphere Client Initialized.")
	// check login
	err = vsphere_api.GlobalClient.LoginViaPassword()
	if err != nil {
		log.Fatalln("Cannot login to vSphere: " + err.Error())
	}
	log.Infoln("Login Successful.")
	defer vsphere_api.GlobalClient.Logout()
	// if not working, detect error and warn user then exit
	err = vsphere_api.GlobalClient.ShowAPIVersion()
	if err != nil {
		log.Fatalln("Connection Check - API Version - Failed: " + err.Error())
	}
	// check current server timestamp
	err = vsphere_api.GlobalClient.CheckTimeSkew()
	if err != nil {
		log.Fatalln("TimeSync Check - Failed: ", err)
	}
	// handle signal
	var sigChan = make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, os.Kill)
	// tasks might be async, especailly download support bundle, must wait
	// max timeout 30mins
	var wgBackground = &sync.WaitGroup{}
	// further cmd
	wgBackground.Add(1)
	go func() {
		defer wgBackground.Done()
		promptPS1 := fmt.Sprintf("[%s @ %s] [>_] $", finalUserInfoInURL.Username(), vcURL.Host)
		// query user input
		for {
			var nextCmd string
			err := survey.AskOne(&survey.Input{
				Message: promptPS1,
				Help:    "Supported commands: [support_bundle] [try_reconnect] [basic_info] [vi_events] [exit] [full_help]",
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
				subcmds.ShowHelp()
				continue
			case "try_reconnect":
				subcmds.TryReconn()
				continue
			case "vi_events":
				subcmds.RetrieveVIEvents()
				continue
			case "support_bundle":
				subcmds.RetrieveSupportBundle()
				continue
			case "basic_info":
				fallthrough
			default:
				fmt.Println("not implemented.")
			}
		}
	}()
	<-sigChan
	log.Println("Exit signal received. Waiting for background tasks. " +
		"Press Ctrl-C again to force exit, but you may experience unexpected data loss.")
	wgBackground.Wait()
	log.Println("Background tasks done. Cleaning up... Exit after 2 seconds.")
	time.Sleep(2 * time.Second)
}
