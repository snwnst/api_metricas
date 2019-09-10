package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	firebase "firebase.google.com/go"
	"github.com/gorilla/mux"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
	"google.golang.org/api/option"
)

func main() {
	go backgrounProsses()
	log.Fatal(http.ListenAndServe(":"+getPortEnvaironment(), getRoutes()))
}

/* CONFIG */

func backgrounProsses() {
	for {
		_hostMetrics := getMetrics()
		ctx := context.Background()
		opt := option.WithCredentialsFile(getPath() + "firebase_key.json")
		config := &firebase.Config{
			ProjectID:   "hostmetrics-cad87",
			DatabaseURL: "https://hostmetrics-cad87.firebaseio.com",
		}
		app, err := firebase.NewApp(ctx, config, opt)
		check(err)
		client, err := app.Database(ctx)
		check(err)
		if err := client.NewRef("hosts/"+_hostMetrics.HostIDUiid).Set(ctx, _hostMetrics); err != nil {
			check(err)
		}
		time.Sleep(5000 * time.Millisecond)
	}
}

func getRoutes() *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc("/prosses/{text}", prossesInNode).Name("prosses").Methods("GET")
	router.HandleFunc("/whrite/{filename}/{text}", whriteStatusNode).Name("whriteInFile").Methods("GET")
	router.HandleFunc("/read/{filename}", readStatusNode).Name("readInFile").Methods("GET")
	router.HandleFunc("/metrics", getMetricsFromNode).Name("getMetricsFromNode").Methods("GET")
	return router
}

func getPortEnvaironment() string {
	port := os.Getenv("PORT")
	if port != "" {
		return port
	}
	return "8085"
}

/* CONTROLLERS */

func whriteStatusNode(w http.ResponseWriter, r *http.Request) {
	response, err := json.Marshal(whriteInFile(mux.Vars(r)["filename"], mux.Vars(r)["text"]))
	check(err)
	fmt.Fprintf(w, string(response))
}

func readStatusNode(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, readInFile(mux.Vars(r)["filename"]))
}

func getMetricsFromNode(w http.ResponseWriter, r *http.Request) {
	response, err := json.Marshal(getMetrics())
	check(err)
	fmt.Fprintf(w, string(response))
}

func prossesInNode(w http.ResponseWriter, r *http.Request) {
	whriteInFile(getFilePath("status"), "PROSESANDO: "+mux.Vars(r)["text"])
	go prossesCia(mux.Vars(r)["text"])
}

/* CORE */

func getMetrics() *hostMetric {

	_hostMetrics := new(hostMetric)
	runtimeOS := runtime.GOOS

	vmStat, err := mem.VirtualMemory()
	check(err)

	diskStat, err := disk.Usage("/")
	check(err)

	cpuStat, err := cpu.Info()
	check(err)

	percentage, err := cpu.Percent(0, true)
	check(err)

	hostStat, err := host.Info()
	check(err)

	interfStat, err := net.Interfaces()
	check(err)

	_hostMetrics.Status = readInFile("status")
	_hostMetrics.Os = runtimeOS
	_hostMetrics.TotalMemory = strconv.FormatUint(vmStat.Total, 10)
	_hostMetrics.FreeMemory = strconv.FormatUint(vmStat.Free, 10)
	_hostMetrics.PercentageUsedMemory = strconv.FormatFloat(vmStat.UsedPercent, 'f', 2, 64)
	_hostMetrics.TotalDiskSpace = strconv.FormatUint(diskStat.Total, 10)
	_hostMetrics.UsedDiskSpace = strconv.FormatUint(diskStat.Used, 10)
	_hostMetrics.FreeDiskDpace = strconv.FormatUint(diskStat.Free, 10)
	_hostMetrics.PercentageDiskSpaceUsage = strconv.FormatFloat(diskStat.UsedPercent, 'f', 2, 64)
	_hostMetrics.CPUCores = strconv.FormatInt(int64(cpuStat[0].Cores), 10)
	_hostMetrics.Hostname = hostStat.Hostname
	_hostMetrics.Uptime = strconv.FormatUint(hostStat.Uptime, 10)
	_hostMetrics.NumbersOfProssesRunning = strconv.FormatUint(hostStat.Procs, 10)
	_hostMetrics.Platform = hostStat.Platform
	_hostMetrics.HostIDUiid = hostStat.HostID

	for _, cpupercent := range percentage {
		_x := cpuNode{}
		_x.CPUIndexNumber = strconv.FormatInt(int64(cpuStat[0].CPU), 10)
		_x.VendorID = cpuStat[0].VendorID
		_x.Family = cpuStat[0].Family
		_x.ModelName = cpuStat[0].ModelName
		_x.Speed = strconv.FormatFloat(cpuStat[0].Mhz, 'f', 2, 64)
		_x.CPUUsedPercentage = strconv.FormatFloat(cpupercent, 'f', 2, 64)
		_hostMetrics.Cores = append(_hostMetrics.Cores, _x)
	}

	for _, interf := range interfStat {
		_iterface := iterface{}
		_iterface.InterfaceName = interf.Name
		_iterface.HardwareMacAddress = interf.HardwareAddr.String()
		for _, flag := range strings.Split(interf.Flags.String(), "|") {
			_iterface.Flags = append(_iterface.Flags, flag)
		}
		addrs, _ := interf.Addrs()
		for _, addr := range addrs {
			_iterface.Ips = append(_iterface.Ips, addr.String())
		}
		_hostMetrics.Interfaces = append(_hostMetrics.Interfaces, _iterface)
	}

	return _hostMetrics
}

func readInFile(filename string) string {
	dat, err := ioutil.ReadFile(getFilePath(filename))
	check(err)
	return string(dat)
}

func whriteInFile(filename string, dataToWhrite string) bool {
	err := ioutil.WriteFile(getFilePath(filename), []byte(dataToWhrite), 0644)
	check(err)
	return true
}

func prossesCia(value string) {
	fmt.Println("ejecuta el .exe con el valor " + value)
	time.Sleep(10000 * time.Millisecond)
	whriteInFile("status", "online")
}

func getPath() string {
	dir, err := os.Getwd()
	check(err)
	return dir + "/"
}

func getFilePath(filename string) string {
	filename = getPath() + filename + ".txt"
	return string(filename)
}

/* ERROR CHECK */

func check(e error) {
	if e != nil {
		panic(e)
	}
}

/* STRUCT MODEL */

type hostMetric struct {
	Status                   string
	Os                       string
	TotalMemory              string
	FreeMemory               string
	PercentageUsedMemory     string
	TotalDiskSpace           string
	UsedDiskSpace            string
	FreeDiskDpace            string
	PercentageDiskSpaceUsage string
	CPUCores                 string
	Hostname                 string
	Uptime                   string
	NumbersOfProssesRunning  string
	Platform                 string
	HostIDUiid               string
	Cores                    []cpuNode
	Interfaces               []iterface
}

type cpuNode struct {
	CPUIndexNumber    string
	VendorID          string
	Family            string
	ModelName         string
	Speed             string
	CPUUsedPercentage string
}

type iterface struct {
	InterfaceName      string
	HardwareMacAddress string
	Flags              []string
	Ips                []string
}
