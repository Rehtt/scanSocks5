package main

import (
	"io"
	"net/http"
	"os"
	"sync"

	"github.com/Rehtt/Kit/util"
	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
)

var (
	regionSeracher     *xdb.Searcher
	regionSeracherInit sync.Once
)

func regionInit() {
	regionSeracherInit.Do(func() {
		info, err := util.GetGitHubFileInfo("zoujingli", "ip2region", "ip2region.xdb")
		if err != nil {
			log.Error("regionSeracherInit check xdb", "err", err)
			return
		}

		downloadDB := func() error {
			log.Info("download ip2region")
			response, err := http.Get(info.DownloadUrl)
			if err != nil {
				return err
			}
			defer response.Body.Close()
			f, err := os.OpenFile(*regionDBPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
			if err != nil {
				return err
			}
			defer f.Close()
			io.Copy(f, response.Body)
			log.Info("download ip2region done")
			return nil
		}

		finfo, err := os.Stat(*regionDBPath)
		if os.IsNotExist(err) || info.Size != int(finfo.Size()) {
			if err := downloadDB(); err != nil {
				log.Error("regionSeracherInit download xdb", "err", err)
				return
			}
		} else if err != nil {
			log.Error("regionSeracherInit open xdb", "err", err)
		}
		regionSeracher, err = xdb.NewWithFileOnly(*regionDBPath)
		if err != nil {
			log.Error("regionSeracherInit NewWithFileOnly", "err", err)
			return
		}
	})
}

func ipRegion(ip string) (string, error) {
	regionInit()
	region, err := regionSeracher.SearchByStr(ip)
	if err != nil {
		log.Error("ipRegion SearchByStr", "err", err)
		return "", err
	}
	return region, nil
}
