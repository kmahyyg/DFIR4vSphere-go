package vsphere_api

import (
	"context"
	"errors"
	"github.com/kmahyyg/DFIR4vSphere-go/pkg/common"
	"github.com/schollz/progressbar/v3"
	log "github.com/sirupsen/logrus"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
	"io"
	"net/http"
	"os"
	"path"
	"sync"
)

var (
	ErrCreateGenerationTaskFailed = errors.New("create task for bundle generation failed")
)

func (vsc *vSphereClient) RequestSupportBundle(hostList []*object.HostSystem, wg *sync.WaitGroup) error {
	if !vsc.postInitDone || !vsc.IsLoggedIn() {
		return ErrSessionInvalid
	}
	progLogger := newDumbProgressLogger("Generating Support Bundle... ")
	diagBundles, err := vsc.generateSupportBundle(hostList, progLogger)
	if err != nil {
		return err
	}
	wg.Add(1)
	err = vsc.downloadSupportBundle(diagBundles, wg)
	if err != nil {
		return err
	}
	return nil
}

func (vsc *vSphereClient) generateSupportBundle(hostList []*object.HostSystem, pLogger *dumbProgressLogger) ([]types.DiagnosticManagerBundleInfo,
	error) {
	// stage 1: generate support bundle
	tmpCtx := context.TODO()
	tasks, err := vsc.vmwDiagMgr.GenerateLogBundles(tmpCtx, true, hostList)
	if err != nil {
		log.Errorln("generate support bundle, internal: ", err)
		return nil, ErrCreateGenerationTaskFailed
	}
	log.Infoln("bundle generator task successfully created. Waiting for generation.")
	log.Infoln("generator task is running, this may take some time, PLEASE WAIT PATIENTLY.")
	// stage 2: show task result
	r, err := tasks.WaitForResult(tmpCtx, pLogger)
	if err != nil {
		return nil, err
	}
	log.Infoln("Bundle successfully generated. Now going to download.")
	return r.Result.(types.ArrayOfDiagnosticManagerBundleInfo).DiagnosticManagerBundleInfo, nil
}

func (vsc *vSphereClient) downloadSupportBundle(bundlesInfo []types.DiagnosticManagerBundleInfo, parentWg *sync.WaitGroup) error {
	// this is used to mark all download tasks are finished.
	defer parentWg.Done()
	// this is used to substantially track download progress
	dwnldTaskWg := &sync.WaitGroup{}
	// implement custom downloader
	// vsc.vmwSoapClient.DownloadFile is just a wrapper of http.Client.Do(ctx, *req)
	for _, v := range bundlesInfo {
		fBundleURL, err := vsc.vmwSoapClient.ParseURL(v.Url)
		if err != nil {
			log.Errorln("parseURL in downloader, err: ", err)
			continue
		}
		dstFile := path.Base(fBundleURL.Path)
		// original default download parameter only consists of GET method definition
		// cmd.DownloadFile -> cmd.client.DownloadFile -> soap.Client.Download -> soap.Client.WriteFile
		dwnldTaskWg.Add(1)
		go progressedDownloader(dstFile, fBundleURL.String(), dwnldTaskWg)
		log.Infoln("downloader task created: ", dstFile)
	}
	log.Infoln("all tasks are downloading, wait until complete.")
	dwnldTaskWg.Wait()
	log.Infoln("all tasks downloaded, finish.")
	return nil
}

func progressedDownloader(dstFile string, url string, dwnldWg *sync.WaitGroup) {
	defer dwnldWg.Done()
	ctx := context.Background()
	httpCli := http.DefaultClient
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Errorln("unknown error occurred when build requests, err: ", err)
		return
	}
	req = req.WithContext(ctx)
	// by default, the param for request build is only GET method, without any cookie
	req.Header.Add("User-Agent", "DFIR4vSphere-Go/"+common.VersionStr)
	resp, err := httpCli.Do(req)
	if err != nil {
		log.Errorln("request sent, response err: ", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		f, err := os.OpenFile(dstFile, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Errorln("cannot write to / create dest file, err: ", err)
			return
		}
		defer f.Close()
		defer f.Sync()
		pgBar := progressbar.DefaultBytes(resp.ContentLength, "downloading "+dstFile+" ...")
		_, err = io.Copy(io.MultiWriter(f, pgBar), resp.Body)
		if err != nil {
			log.Errorf("error while downloading %s from network: %v", dstFile, err)
			return
		}
	} else {
		log.Errorln("resp code not 200, currently: ", resp.StatusCode)
		return
	}
}
