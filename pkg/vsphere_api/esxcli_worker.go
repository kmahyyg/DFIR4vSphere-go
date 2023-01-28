package vsphere_api

import (
	"encoding/csv"
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"github.com/vmware/govmomi/govc/host/esxcli"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	esxCLIcmdLst = map[string]string{
		"SysAcct":         "system account list",
		"SysPerm":         "system permission list",
		"SysMod":          "system module list",
		"SysProc":         "system process list",
		"SysPrefEnc":      "system settings encryption get",
		"SysPrefGuestSt":  "system settings gueststore repository get",
		"SysPrefAdvList":  "system settings advanced list",
		"SysPrefKernel":   "system settings kernel list",
		"SysLogConf":      "system syslog config get",
		"SysSecurityCert": "system security certificatestore list",
		"SoftVIBVerify":   "software vib signature verify",
		"SoftBaseImg":     "software baseimage get",
		"SoftVIBs":        "software vib get",
		"SoftProfiles":    "software profile get",
		"StorFS":          "storage filesystem list",
		"StorIOFilters":   "storage iofilter list",
		"NetIPConn":       "network ip connection list",
		"NetIPv6Nic":      "network ip interface ipv6 get",
		"NetIPv4Nic":      "network ip interface ipv4 get",
		"NetIPv6Route":    "network ip route ipv6 list",
		"NetIPv4Route":    "network ip route ipv4 list",
		"NetVMList":       "network vm list",
		"NetIPARPCache":   "network ip neighbor list",
	}
)

func (esxhbi *ESXHostBasicInfo) GetInfoFunc2() (err error) {
	// esxcli must be run using real esxi instance, the simulator does NOT implement necessary method
	if esxhbi.esxcliExec == nil {
		return ErrPrerequisitesNotSatisfied
	}
	// execute command list, command are recorded in docs
	// save as csv file, filename: machineName-CommandName-Timestamp.csv
	// optimize output file location, must be saved to seperator folder, named by current date using YYYYMMDD
	var machineName = filepath.Base(esxhbi.InventoryPath)
	if len(machineName) == 0 {
		machineName, err = GetNanoID(6)
		if err != nil {
			return err
		}
	}
	for k, v := range esxCLIcmdLst {
		resp, err := esxhbi.esxcliExec.Run(strings.Split(v, " "))
		if err != nil {
			log.Errorln("ESXCLI Exec -", k, ", Err: ", err)
			continue
		}
		err = FormatAndSave(machineName, k, resp)
		if err != nil {
			log.Errorln("ESXCLI Format and Save -", k, " Err:", err)
			continue
		}
	}
	return nil
}

func FormatAndSave(machineName string, cateName string, resp *esxcli.Response) (err error) {
	var formatType string
	if resp.Info != nil {
		formatType = resp.Info.Hints.Formatter()
	}
	// sort data before save
	var fieldKeys []string
	// create and save
	var fDstPath = "output/" + machineName + "-" + cateName + "-" + strconv.FormatInt(time.Now().Unix(), 10)
	// create corresponding writer
	var alreadyTabled bool
	var fd *os.File
	switch formatType {
	case "table":
		// use csv
		fieldKeys = resp.Info.Hints.Fields()
		alreadyTabled = true
		fallthrough
	default:
		// use json
		if alreadyTabled && len(fieldKeys) != 0 {
			fDstPath += ".csv"
		} else {
			fDstPath += ".json"
		}
		// detect if table failed or not initialized
		if len(fieldKeys) == 0 {
			alreadyTabled = false
			// only get first result to append field key
			for key := range resp.Values[0] {
				fieldKeys = append(fieldKeys, key)
			}
		}
		// sort to make sure all in stable order
		sort.Strings(fieldKeys)
		// create file
		fd, err = os.Create(fDstPath)
		if err != nil {
			return err
		}
		defer fd.Close()
		defer fd.Sync()
		// response value are always in key-value or header-data format,
		// govc/host/esxcli/response.go:24   type Values map[string][]string
		if alreadyTabled {
			cwr := csv.NewWriter(fd)
			err = cwr.Write(fieldKeys)
			if err != nil {
				log.Errorln("csv write error:", err)
				return err
			}
			for _, sv := range resp.Values {
				// each esxcli.Value is a result
				dt := make([]string, 0)
				for _, k := range fieldKeys {
					if val, ok := sv[k]; ok {
						dt = append(dt, strings.Join(val, "AND"))
					} else {
						dt = append(dt, " ")
					}
				}
				err = cwr.Write(dt)
				if err != nil {
					log.Errorln("csv write error:", err)
					continue
				}
			}
			cwr.Flush()
		} else {
			type resWrapper struct {
				Data []esxcli.Values `json:"data,omitempty"`
			}
			rwer := &resWrapper{Data: resp.Values}
			respData, err := json.Marshal(rwer)
			if err != nil {
				return err
			}
			_, err = fd.Write(respData)
			if err != nil {
				return err
			}
			return nil
		}
	}
	return nil
}
