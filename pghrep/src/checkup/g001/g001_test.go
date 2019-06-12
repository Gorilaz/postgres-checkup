package g001

import (
	"fmt"
	"testing"

	checkup ".."
	"../a001"
)

var TestLastNodesJson checkup.ReportLastNodes = checkup.ReportLastNodes{
	Hosts: checkup.ReportHosts{
		"test-host": {
			Role: "master",
		},
	},
}

func TestG001Success(t *testing.T) {
	fmt.Println(t.Name())
	// G001
	var report G001Report
	var hostResult G001ReportHostResult

	hostResult.Data = map[string]G001Setting{
		"shared_buffers": G001Setting{
			Name:    "shared_buffers",
			Setting: "4194304",
			Unit:    "8kB",
		},
	}
	report.Results = G001ReportHostsResults{"test-host": hostResult}

	// A001
	var a001Report a001.A001Report
	var a001HostResult a001.A001ReportHostResult
	a001HostResult.Data = a001.A001ReportHostResultData{
		Ram: a001.A001ReportRam{
			MemTotal: "65888240 kB",
		},
	}
	a001Report.Results = a001.A001ReportHostsResults{"test-host": a001HostResult}

	result := G001Process(report, a001Report)

	if result.P1 || result.P2 || result.P3 &&
		checkup.ResultInList(result.Conclusions, G001_SHARED_BUFFERS_NOT_OPTIMAL) {
		t.Fatal()
	}

	checkup.PrintResultConclusions(result)
	checkup.PrintResultRecommendations(result)
}

func TestG001SharedBuffersLow(t *testing.T) {
	fmt.Println(t.Name())
	// G001
	var report G001Report
	var hostResult G001ReportHostResult

	hostResult.Data = map[string]G001Setting{
		"shared_buffers": G001Setting{
			Name:    "shared_buffers",
			Setting: "1235404",
			Unit:    "8kB",
		},
	}
	report.Results = G001ReportHostsResults{"test-host": hostResult}

	// A001
	var a001Report a001.A001Report
	var a001HostResult a001.A001ReportHostResult
	a001HostResult.Data = a001.A001ReportHostResultData{
		Ram: a001.A001ReportRam{
			MemTotal: "65888240 kB",
		},
	}
	a001Report.Results = a001.A001ReportHostsResults{"test-host": a001HostResult}

	result := G001Process(report, a001Report)

	if !result.P1 ||
		!checkup.ResultInList(result.Conclusions, G001_SHARED_BUFFERS_NOT_OPTIMAL) ||
		!checkup.ResultInList(result.Recommendations, G001_SHARED_BUFFERS_NOT_OPTIMAL) ||
		!checkup.ResultInList(result.Recommendations, G001_TUNE_SHARED_BUFFERS) {
		t.Fatal()
	}

	checkup.PrintResultConclusions(result)
	checkup.PrintResultRecommendations(result)
}

func TestG001SharedBuffersHigh(t *testing.T) {
	fmt.Println(t.Name())
	// G001
	var report G001Report
	var hostResult G001ReportHostResult

	hostResult.Data = map[string]G001Setting{
		"shared_buffers": G001Setting{
			Name:    "shared_buffers",
			Setting: "6753544",
			Unit:    "8kB",
		},
	}
	report.Results = G001ReportHostsResults{"test-host": hostResult}

	// A001
	var a001Report a001.A001Report
	var a001HostResult a001.A001ReportHostResult
	a001HostResult.Data = a001.A001ReportHostResultData{
		Ram: a001.A001ReportRam{
			MemTotal: "65888240 kB",
		},
	}
	a001Report.Results = a001.A001ReportHostsResults{"test-host": a001HostResult}

	result := G001Process(report, a001Report)

	if !result.P1 ||
		!checkup.ResultInList(result.Conclusions, G001_SHARED_BUFFERS_NOT_OPTIMAL) ||
		!checkup.ResultInList(result.Recommendations, G001_SHARED_BUFFERS_NOT_OPTIMAL) ||
		!checkup.ResultInList(result.Recommendations, G001_TUNE_SHARED_BUFFERS) {
		t.Fatal()
	}

	checkup.PrintResultConclusions(result)
	checkup.PrintResultRecommendations(result)
}

func TestG001OOM(t *testing.T) {
	fmt.Println(t.Name())
	// G001
	var report G001Report
	var hostResult G001ReportHostResult

	hostResult.Data = map[string]G001Setting{
		"shared_buffers": G001Setting{
			Name:    "shared_buffers",
			Setting: "4194304",
			Unit:    "8kB",
		},
		"autovacuum_max_workers": G001Setting{
			Name:    "autovacuum_max_workers",
			Setting: "10",
			Unit:    "",
		},
		"autovacuum_work_mem": G001Setting{
			Name:    "autovacuum_work_mem",
			Setting: "-1",
			Unit:    "kB",
		},
		"effective_cache_size": G001Setting{
			Name:    "effective_cache_size",
			Setting: "6291456",
			Unit:    "8kB",
		},
		"maintenance_work_mem": G001Setting{
			Name:    "maintenance_work_mem",
			Setting: "2097152",
			Unit:    "kB",
		},
		"max_connections": G001Setting{
			Name:    "max_connections",
			Setting: "1000",
			Unit:    "",
		},
		"temp_buffers": G001Setting{
			Name:    "temp_buffers",
			Setting: "8192",
			Unit:    "8kB",
		},
		"work_mem": G001Setting{
			Name:    "work_mem",
			Setting: "65536",
			Unit:    "kB",
		},
	}
	report.Results = G001ReportHostsResults{"test-host": hostResult}
	report.LastNodesJson = TestLastNodesJson

	// A001
	var a001Report a001.A001Report
	var a001HostResult a001.A001ReportHostResult
	a001HostResult.Data = a001.A001ReportHostResultData{
		Ram: a001.A001ReportRam{
			MemTotal: "65888240 kB",
		},
	}
	a001Report.Results = a001.A001ReportHostsResults{"test-host": a001HostResult}

	result := G001Process(report, a001Report)

	if !result.P1 || !checkup.ResultInList(result.Conclusions, G001_OOM) {
		t.Fatal()
	}

	checkup.PrintResultConclusions(result)
	checkup.PrintResultRecommendations(result)
}
