package bt

import (
	"context"
	"errors"
	"math"
	"strconv"
	"strings"
)

var ClocksPerSec = float64(100)

const (
	MAX_PERCENTAGE   = 100
	MIN_PERCENTAGE   = 0
)

type TimesStat struct {
	CPU       string  `json:"cpu"`
	User      float64 `json:"user"`
	System    float64 `json:"system"`
	Idle      float64 `json:"idle"`
	Nice      float64 `json:"nice"`
	Iowait    float64 `json:"iowait"`
	Irq       float64 `json:"irq"`
	Softirq   float64 `json:"softirq"`
	Steal     float64 `json:"steal"`
	Guest     float64 `json:"guest"`
	GuestNice float64 `json:"guestNice"`
	Offline   float64 `json:"offline"`
}

func Times(percpu bool) ([]TimesStat, error) {
	return TimesWithContext(context.Background(), percpu)
}

func TimesWithContext(ctx context.Context, percpu bool) ([]TimesStat, error) {
	filename := HostProc("bt_stat")
	var lines = []string{}
	if percpu {
		statlines, err := ReadLines(filename)
		if err != nil || len(statlines) < 2 {
			return []TimesStat{}, nil
		}
		for _, line := range statlines[1:] {
			if !strings.HasPrefix(line, "cpu") {
				break
			}
			lines = append(lines, line)
		}
	} else {
		lines, _ = ReadLinesOffsetN(filename, 0, 1)
	}

	ret := make([]TimesStat, 0, len(lines))

	for _, line := range lines {
		ct, err := parseStatLine(line)
		if err != nil {
			continue
		}
		ret = append(ret, *ct)

	}
	return ret, nil
}

//copy from "github.com/shirou/gopsutil/cpu"
func parseStatLine(line string) (*TimesStat, error) {
	fields := strings.Fields(line)

	if len(fields) == 0 {
		return nil, errors.New("stat does not contain cpu info")
	}

	if strings.HasPrefix(fields[0], "cpu") == false {
		return nil, errors.New("not contain cpu")
	}

	cpu := fields[0]
	if cpu == "cpu" {
		cpu = "cpu-total"
	}
	user, err := strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return nil, err
	}
	nice, err := strconv.ParseFloat(fields[2], 64)
	if err != nil {
		return nil, err
	}
	system, err := strconv.ParseFloat(fields[3], 64)
	if err != nil {
		return nil, err
	}
	idle, err := strconv.ParseFloat(fields[4], 64)
	if err != nil {
		return nil, err
	}
	iowait, err := strconv.ParseFloat(fields[5], 64)
	if err != nil {
		return nil, err
	}
	irq, err := strconv.ParseFloat(fields[6], 64)
	if err != nil {
		return nil, err
	}
	softirq, err := strconv.ParseFloat(fields[7], 64)
	if err != nil {
		return nil, err
	}
	offline, err := strconv.ParseFloat(fields[len(fields)-1], 64)
	if err != nil {
		return nil, err
	}

	ct := &TimesStat{
		CPU:     cpu,
		User:    user / ClocksPerSec,
		Nice:    nice / ClocksPerSec,
		System:  system / ClocksPerSec,
		Idle:    idle / ClocksPerSec,
		Iowait:  iowait / ClocksPerSec,
		Irq:     irq / ClocksPerSec,
		Softirq: softirq / ClocksPerSec,
		Offline: offline / ClocksPerSec,
	}
	if len(fields) > 9 { // Linux >= 2.6.11
		steal, err := strconv.ParseFloat(fields[8], 64)
		if err != nil {
			return nil, err
		}
		ct.Steal = steal / ClocksPerSec
	}
	if len(fields) > 10 { // Linux >= 2.6.24
		guest, err := strconv.ParseFloat(fields[9], 64)
		if err != nil {
			return nil, err
		}
		ct.Guest = guest / ClocksPerSec
	}
	if len(fields) > 11 { // Linux >= 3.2.0
		guestNice, err := strconv.ParseFloat(fields[10], 64)
		if err != nil {
			return nil, err
		}
		ct.GuestNice = guestNice / ClocksPerSec
	}

	return ct, nil
}

func (c TimesStat) String() string {
	v := []string{
		`"cpu":"` + c.CPU + `"`,
		`"user":` + strconv.FormatFloat(c.User, 'f', 1, 64),
		`"system":` + strconv.FormatFloat(c.System, 'f', 1, 64),
		`"idle":` + strconv.FormatFloat(c.Idle, 'f', 1, 64),
		`"nice":` + strconv.FormatFloat(c.Nice, 'f', 1, 64),
		`"iowait":` + strconv.FormatFloat(c.Iowait, 'f', 1, 64),
		`"irq":` + strconv.FormatFloat(c.Irq, 'f', 1, 64),
		`"softirq":` + strconv.FormatFloat(c.Softirq, 'f', 1, 64),
		`"steal":` + strconv.FormatFloat(c.Steal, 'f', 1, 64),
		`"guest":` + strconv.FormatFloat(c.Guest, 'f', 1, 64),
		`"guestNice":` + strconv.FormatFloat(c.GuestNice, 'f', 1, 64),
		`"offline":` + strconv.FormatFloat(c.Offline, 'f', 1, 64),
	}

	return `{` + strings.Join(v, ",") + `}`
}

//copy from "github.com/shirou/gopsutil/cpu"
// Total returns the total number of seconds in a CPUTimesStat
func (c TimesStat) Total() float64 {
	total := c.User + c.System + c.Nice + c.Iowait + c.Irq + c.Softirq +
		c.Steal + c.Idle
	return total
}

func (c TimesStat)CalculateBtOffline(stat2 TimesStat) (float64, float64) {
	stat1All, stat1Offline := c.getAllOffline()
	stat2All, stat2Offline := stat2.getAllOffline()
	if stat2Offline <= stat1Offline {
		return 0, MIN_PERCENTAGE
	}
	if stat2All <= stat1All {
		return stat2Offline - stat1Offline, MAX_PERCENTAGE
	}

	return stat2Offline - stat1Offline, math.Min(MAX_PERCENTAGE, math.Max(MIN_PERCENTAGE, (stat2Offline-stat1Offline)/(stat2All-stat1All)*100))
}

func (c TimesStat)getAllOffline() (float64, float64) {
	busy := c.User + c.System + c.Nice + c.Iowait + c.Irq + c.Softirq + c.Steal
	return busy + c.Idle, c.Offline
}
