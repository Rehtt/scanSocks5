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
	"sync/atomic"
	"time"

	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
)

var Version string = "0.1.1"

var (
	infile      = flag.String("i", "./ip.txt", "ip file, input - linux pipeline will be used")
	usePipeline bool
	limit       = flag.Int("limit", 5, "thread limit")
	ports       []string
	portsStr    = flag.String("ports", "7890,7891,7892,7893,10810", "ports")

	region             = flag.String("region", "", "filter region")
	regionSeracher     *xdb.Searcher
	regionSeracherInit sync.Once
	regionDBPath       = flag.String("region_db_path", "./ip2region.xdb", "region db path")

	count atomic.Int32

	q   = flag.Bool("q", false, "quiet log")
	log *slog.Logger
)

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

	if *q {
		log = newLogger(slog.LevelError)
	} else {
		log = newLogger(slog.LevelInfo)
	}

	inputFile := *infile
	usePipeline = *infile == "-"
	if usePipeline {
		inputFile = "pipeline"
	}

	log.Info("info", "version", Version, "home page", "https://github.com/Rehtt/scanSocks5")
	log.Info("config", "thread limit", *limit, "input file", inputFile, "ports", ports, "filter region", *region)

	var rawData io.Reader
	if usePipeline {
		rawData = os.Stdin
	} else {
		f, err := os.Open(*infile)
		if err != nil {
			log.Error("open file", "err", err)
			return
		}
		defer f.Close()
		rawData = f
	}
	thread := NewThread(*limit)
	buf := bufio.NewScanner(rawData)
	for buf.Scan() {
		ip := strings.TrimSpace(buf.Text())
		if ip == "" {
			continue
		}
		handleScan(thread, ip)
	}
	thread.Wait()
	slog.Info("done", "count", count.Load())
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
				count.Add(1)
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
			log.Info("download ip2region")
			response, err := http.Get("https://github.com/zoujingli/ip2region/raw/master/ip2region.xdb")
			if err != nil {
				log.Error("regionSeracherInit download xdb", "err", err)
				return
			}
			defer response.Body.Close()
			f, err := os.OpenFile(*regionDBPath, os.O_WRONLY|os.O_CREATE, 0644)
			if err != nil {
				log.Error("regionSeracherInit write", "err", err)
				return
			}
			defer f.Close()
			io.Copy(f, response.Body)
		}
		regionSeracher, err = xdb.NewWithFileOnly(*regionDBPath)
		if err != nil {
			log.Error("regionSeracherInit NewWithFileOnly", "err", err)
			return
		}
	})
	region, err := regionSeracher.SearchByStr(ip)
	if err != nil {
		log.Error("ipRegion SearchByStr", "err", err)
		return "", err
	}
	return region, nil
}
