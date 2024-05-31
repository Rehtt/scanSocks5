package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

var Version string = "0.1.1"

var (
	infile  = flag.String("i", "./ip.txt", "ip file, input - linux pipeline will be used")
	outfile = flag.String("o", "", "result output file")
	input   io.Reader
	output  io.Writer

	limit    = flag.Int("limit", 5, "thread limit")
	ports    []string
	portsStr = flag.String("ports", "7890,7891,7892,7893,10810", "ports")

	region        = flag.String("region", "", "filter region")
	excludeRegion = flag.String("exclude_region", "", "exclude region")
	regionDBPath  = flag.String("region_db_path", "./ip2region.xdb", "region db path")

	connectTimeout  = flag.Int("connect_timeout", 2, "ip connect timeout")
	connectDeadline = flag.Int("connect_deadline", 1, "connect read write timeout")

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
	if inputFile == "-" {
		inputFile = "pipeline"
		input = os.Stdin
	} else {
		f, err := os.Open(*infile)
		if err != nil {
			log.Error("open input file", "err", err)
			return
		}
		defer f.Close()
		input = f
	}
	outputFile := *outfile
	if outputFile == "" || outputFile == "-" {
		outputFile = "print"
		output = os.Stdout
	} else {
		f, err := os.OpenFile(outputFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC|os.O_SYNC, 0644)
		if err != nil {
			log.Error("open output file", "err", err)
			return
		}
		defer f.Close()
		output = f
	}

	log.Info("info", "Version", Version, "HomePage", "https://github.com/Rehtt/scanSocks5")
	log.Info("config", "ThreadLimit", *limit, "Input", inputFile, "Ports", ports, "FilterRegion", *region, "Output", outputFile)

	thread := NewThread(*limit)
	buf := bufio.NewScanner(input)
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
	if *excludeRegion != "" && strings.Contains(r, *excludeRegion) {
		return
	}
	for _, port := range ports {
		t.Run(func(arg map[string]any) {
			ip := arg["ip"].(string)
			port := arg["port"].(string)
			if scan(ip, port) {
				fmt.Fprintln(output, ip, port, r)
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
	n, err := net.DialTimeout("tcp", addr, time.Second*time.Duration(*connectTimeout))
	if err != nil {
		return false
	}
	n.SetDeadline(time.Now().Add(time.Second * time.Duration(*connectDeadline)))
	defer n.Close()
	n.Write([]byte("\x05\x01\x00"))
	tmp := make([]byte, 2)
	n.Read(tmp)
	if tmp[0] == 5 {
		return true
	}
	return false
}
