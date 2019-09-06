package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	firebase "firebase.google.com/go"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
	"google.golang.org/api/option"

	"github.com/gorilla/mux"
)

func prosses(w http.ResponseWriter, r *http.Request) {
	whriteInFile("status", "PROSESANDO: "+mux.Vars(r)["text"])
	go subProsses(mux.Vars(r)["text"])
}

func whrite(w http.ResponseWriter, r *http.Request) {
	whriteInFile(mux.Vars(r)["filename"], mux.Vars(r)["text"])
}

func read(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, readInFile(mux.Vars(r)["filename"]))
}

func metrics(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, getMetrics())
}

func subProsses(valor string) {
	fmt.Println("ejecuta el .exe con el valor" + valor)
	time.Sleep(10000 * time.Millisecond)
	whriteInFile("status", "online")
}

func getMetrics() string {
	runtimeOS := runtime.GOOS

	// memory
	vmStat, err := mem.VirtualMemory()
	check(err)

	diskStat, err := disk.Usage("/")
	check(err)

	// cpu - get CPU number of cores and speed
	cpuStat, err := cpu.Info()
	check(err)
	percentage, err := cpu.Percent(0, true)
	check(err)

	// host or machine kernel, uptime, platform Info
	hostStat, err := host.Info()
	check(err)

	// get interfaces MAC/hardware address
	interfStat, err := net.Interfaces()
	check(err)

	_hostMetrics := new(hostMetrics)
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
		_x := core{}
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
	urlsJSON, _ := json.Marshal(_hostMetrics)
	//whriteInFile("specs.json", string(urlsJSON))
	//println(string(_hostMetrics.Status))

	ctx := context.Background()
	opt := option.WithCredentialsFile("hostmetrics-cad87-firebase-adminsdk-pqrth-91458478e6.json")
	config := &firebase.Config{
		ProjectID:   "hostmetrics-cad87",
		DatabaseURL: "https://hostmetrics-cad87.firebaseio.com",
	}
	app, err := firebase.NewApp(ctx, config, opt)
	if err != nil {
		log.Fatal(err)
	}

	client, err := app.Database(ctx)
	if err != nil {
		log.Fatal(err)
	}

	if err := client.NewRef("hosts/"+_hostMetrics.HostIDUiid).Set(ctx, _hostMetrics); err != nil {
		log.Fatal(err)
	}

	return string(urlsJSON)

}

func checkExpire() {
	for {
		getMetrics()
		time.Sleep(5000 * time.Millisecond)
	}
}

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/prosses/{text}", prosses).Name("prosses").Methods("GET")
	router.HandleFunc("/whrite/{filename}/{text}", whrite).Name("whrite").Methods("GET")
	router.HandleFunc("/read/{filename}", read).Name("read").Methods("GET")
	router.HandleFunc("/metrics", metrics).Name("cpuMetrics").Methods("GET")
	go checkExpire()
	http.ListenAndServe(":8080", router)
}

func whriteInFile(filename string, dataToWhrite string) {
	err := ioutil.WriteFile(filename, []byte(dataToWhrite), 0644)
	check(err)
}

func readInFile(filename string) string {
	dat, err := ioutil.ReadFile(filename)
	check(err)
	return string(dat)
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

type core struct {
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

type hostMetrics struct {
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
	Cores                    []core
	Interfaces               []iterface
}
