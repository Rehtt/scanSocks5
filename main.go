package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
)

var (
	pipe     = flag.Bool("pipeline", false, "use linux pipeline")
	limit    = flag.Int("limit", 5, "thread limit")
	ports    []string
	portsStr = flag.String("ports", "7890,7891,7892,7893,10810", "ports")

	region             = flag.String("region", "", "filter region")
	regionSeracher     *xdb.Searcher
	regionSeracherInit sync.Once
	regionDBPath       = flag.String("region_db_path", "./ip2region.xdb", "region db path")
)

type Thread struct {
	c  chan struct{}
	wg sync.WaitGroup
}

func NewThread(limit int) *Thread {
	return &Thread{
		c: make(chan struct{}, limit),
	}
}

func (t *Thread) Run(f func(map[string]any), arg map[string]any) {
	t.c <- struct{}{}
	t.wg.Add(1)
	go func(t *Thread, f func(arg map[string]any), arg map[string]any) {
		f(arg)
		<-t.c
		t.wg.Done()
	}(t, f, arg)
}

func (t *Thread) Wait() {
	t.wg.Wait()
}

func main() {
	flag.Parse()
	for _, v := range strings.Split(*portsStr, ",") {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		_, err := strconv.Atoi(v)
		if err == nil {
			ports = append(ports, v)
		}
	}
	slog.Info("config", "thread limit", *limit, "pipeline", *pipe, "ports", ports, "filter region", *region)

	if *pipe {
		pipeHandle()
		return
	}
	fileHandle()
}

func fileHandle() {
	ipRaw, err := os.ReadFile("ip.txt")
	if err != nil {
		slog.Error("open file", "err", err)
		return
	}
	ips := strings.Split(string(ipRaw), "\n")
	slog.Info("ips", "len", len(ips))
	thread := NewThread(*limit)
	for _, ip := range ips {
		ip = strings.TrimSpace(ip)
		if ip == "" {
			continue
		}
		handleScan(thread, ip)
	}
	thread.Wait()
}

func pipeHandle() {
	buf := bufio.NewScanner(os.Stdin)
	thread := NewThread(*limit)
	for buf.Scan() {
		txt := strings.TrimSpace(buf.Text())
		if txt == "" {
			continue
		}
		handleScan(thread, txt)
	}
	thread.Wait()
}

func handleScan(t *Thread, ip string) {
	r, err := ipRegion(ip)
	if err != nil {
		panic(err)
	}
	if !strings.Contains(r, *region) {
		return
	}
	for _, port := range ports {
		t.Run(func(arg map[string]any) {
			ip := arg["ip"].(string)
			port := arg["port"].(string)
			if scan(ip, port) {
				fmt.Println(ip, port, r)
			}
		}, map[string]any{
			"ip":   ip,
			"port": port,
		})
	}
}

func scan(ip, port string) bool {
	addr := ip + ":" + port
	n, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return false
	}
	n.SetDeadline(time.Now().Add(1 * time.Second))
	defer n.Close()
	n.Write([]byte("\x05\x01\x00"))
	tmp := make([]byte, 2)
	n.Read(tmp)
	if tmp[0] == 5 {
		return true
	}
	return false
}

func ipRegion(ip string) (string, error) {
	regionSeracherInit.Do(func() {
		_, err := os.Stat(*regionDBPath)
		if os.IsNotExist(err) {
			slog.Info("download ip2region")
			response, err := http.Get("https://github.com/zoujingli/ip2region/raw/master/ip2region.xdb")
			if err != nil {
				slog.Error("regionSeracherInit download xdb", "err", err)
				return
			}
			defer response.Body.Close()
			f, err := os.OpenFile(*regionDBPath, os.O_WRONLY|os.O_CREATE, 0644)
			if err != nil {
				slog.Error("regionSeracherInit write", "err", err)
				return
			}
			defer f.Close()
			io.Copy(f, response.Body)
		}
		regionSeracher, err = xdb.NewWithFileOnly(*regionDBPath)
		if err != nil {
			slog.Error("regionSeracherInit NewWithFileOnly", "err", err)
			return
		}
	})
	region, err := regionSeracher.SearchByStr(ip)
	if err != nil {
		slog.Error("ipRegion SearchByStr", "err", err)
		return "", err
	}
	return region, nil
}
