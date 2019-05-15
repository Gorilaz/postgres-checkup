package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"../src/checkup"
	"../src/pyraconv"
)

const CRITICAL_USAGE int = 90
const PROBLEM_USAGE int = 70

var VALID_FS []string = []string{
	"ext4",
	"xfs",
	"tmpfs",
}

type prepare string

type FsItem struct {
	Fstype     string `json:"fstype"`
	Size       string `json:"size"`
	Avail      string `json:"avail"`
	Used       string `json:"used"`
	UsePercent string `json:"use_percent"`
	MountPoint string `json:"mount_point"`
	Path       string `json:"path"`
	Device     string `json:"device"`
}

type ReportHostResultData struct {
	DbData map[string]FsItem `json:"db_data"`
	FsData map[string]FsItem `json:"fs_data"`
}

type ReportHostResult struct {
	Data      ReportHostResultData    `json:"data"`
	NodesJson checkup.ReportLastNodes `json:"nodes.json"`
}

type ReportHostsResults map[string]ReportHostResult

type Report struct {
	Project       string                  `json:"project"`
	Name          string                  `json:"name"`
	CheckId       string                  `json:"checkId"`
	Timestamptz   string                  `json:"timestamptz"`
	Database      string                  `json:"database"`
	Dependencies  map[string]interface{}  `json:"dependencies"`
	LastNodesJson checkup.ReportLastNodes `json:"last_nodes_json"`
	Results       ReportHostsResults      `json:"results"`
}

func readData(filePath string) *Report {
	file, err := os.Open(filePath)
	if err != nil {
		return nil
	}
	defer file.Close()
	jsonRaw, err := ioutil.ReadAll(file)
	if err != nil {
		return nil
	}
	var report Report
	err = json.Unmarshal(jsonRaw, &report)
	if err != nil {
		return nil
	}
	return &report
}

func isValidFs(fs string) bool {
	for _, validFs := range VALID_FS {
		if validFs == fs {
			return true
		}
	}
	return false
}

func checkFsItem(host string, fsItemData FsItem,
	conclusions []string, recommendations []string) (bool, bool, bool) {
	nfs := false
	less70p := false
	nonExt4 := false

	if isValidFs(strings.ToLower(fsItemData.Fstype)) != true {
		nonExt4 = true
	}
	if strings.ToLower(fsItemData.Fstype[0:3]) == "nfs" {
		nfs = true
	}
	usePercent := strings.Replace(fsItemData.UsePercent, "%", "", 1)
	percent, _ := strconv.Atoi(usePercent)
	if percent < PROBLEM_USAGE {
		less70p = true
	}
	if percent >= PROBLEM_USAGE && percent < CRITICAL_USAGE {
		conclusions = append(conclusions, fmt.Sprintf("[P2] Disk %s on %s space usage is %s, it exceeds 70%%. "+
			"There are some risks of out-of-disk-space problem.", fsItemData.MountPoint, host, fsItemData.Used))
		recommendations = append(recommendations, fmt.Sprintf("[P2] Add more disk %s on %s space. "+
			"It is recommended to keep free disk space less than 70%%. "+
			"To reduce risks of out-of-disk-space problem.", fsItemData.MountPoint, host))
	}
	if percent >= CRITICAL_USAGE {
		conclusions = append(conclusions, fmt.Sprintf("Disk %s on %s space usage is %s, it exceeds 90%%. "+
			"There are significant risks of out-of-disk-space problem. "+
			"In this case, PostgreSQL will stop working and manual fix will be required.",
			fsItemData.MountPoint, host, fsItemData.Used))
		recommendations = append(recommendations, fmt.Sprintf("[P1] Add more disk %s on %s space as "+
			"soon as possible to prevent outage.", fsItemData.MountPoint, host))

	}
	return less70p, nfs, nonExt4
}

// Generate conclusions and recommendatons
func A008(data map[string]interface{}) {
	//	nfs := false
	less70p := false
	p1 := false
	p2 := false
	p3 := false
	var nfsConclusions []string
	var nExtConclusions []string
	var conclusions []string
	var recommendations []string
	filePath := pyraconv.ToString(data["source_path_full"])
	report := readData(filePath)
	if report == nil {
		return
	}
	for host, hostResult := range report.Results {
		var nfsItems []FsItem
		var notExtItems []FsItem
		for _, fsItemData := range hostResult.Data.DbData {
			l, n, ne := checkFsItem(host, fsItemData, conclusions, recommendations)
			less70p = less70p || l
			if n {
				nfsItems = append(nfsItems, fsItemData)
			}
			if ne {
				notExtItems = append(notExtItems, fsItemData)
			}
		}
		for _, fsItemData := range hostResult.Data.FsData {
			l, _, _ := checkFsItem(host, fsItemData, conclusions, recommendations)
			less70p = less70p || l
		}
		if len(nfsItems) > 0 {
			var nfsDisks []string
			for _, nfsItem := range nfsItems {
				nfsDisks = append(nfsDisks, nfsItem.MountPoint)
			}
			p1 = true
			areIs := "is"
			if len(nfsDisks) > 1 {
				areIs = "are"
			}
			nfsConclusions = append(nfsConclusions, fmt.Sprintf("[P1] %s on %s "+areIs+" located on an NFS drive. "+
				"This might lead to serious issues with Postgres, including downtime and data corruption.",
				strings.Join(nfsDisks, ", "), host))
		}
		if len(notExtItems) > 0 {
			var nExtDisks []string
			var nExtDiskFs []string
			for _, nExtItem := range notExtItems {
				nExtDisks = append(nExtDisks, nExtItem.MountPoint)
				nExtDiskFs = append(nExtDiskFs, nExtItem.Fstype)
			}
			p3 = true
			areIs := "is"
			respectively := ""
			s := ""
			if len(nExtDisks) > 1 {
				areIs = "are"
				respectively = " respectively"
				s = "s"
			}
			nExtConclusions = append(nExtConclusions, fmt.Sprintf("[P3] %s on %s "+areIs+
				" located on drive"+s+" where the following filesystems are used: %s"+respectively+
				". This might mean that Postgres performance and reliability characteristics are worse than it "+
				"could be in case of use of more popular filesystems (such as ext4).",
				strings.Join(nExtDisks, ", "), host, strings.Join(nExtDiskFs, ", ")))
		}

	}
	if less70p && len(recommendations) == 0 {
		conclusions = append(conclusions, "No significant risks of out-of-disk-space problem have been detected.")
	}
	if len(nfsConclusions) > 0 {
		conclusions = append(conclusions, nfsConclusions...)
		recommendations = append(recommendations, "[P1] Do not use NFS for Postgres.")
	}
	if len(nExtConclusions) > 0 {
		conclusions = append(conclusions, nExtConclusions...)
		recommendations = append(recommendations, "[P3] Consider using ext4 for all Postgres directories.")
	}
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "No recommendations.")
	}

	// update data
	data["conclusions"] = conclusions
	data["recommendations"] = recommendations
	data["p1"] = p1
	data["p2"] = p2
	data["p3"] = p3
	// update file
	checkup.SaveJsonConclusionsRecommendations(data, conclusions, recommendations, p1, p2, p3)
}

// Plugin entry point
func (g prepare) Prepare(data map[string]interface{}) map[string]interface{} {
	A008(data)
	return data
}

var Preparer prepare
