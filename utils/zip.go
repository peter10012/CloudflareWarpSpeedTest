package utils

import (
	"archive/zip"
	"fmt"
	"io"
	"os"

	"gopkg.in/ini.v1"
)

const (
	defaultExportWireguardZip = true
	defaultOutputZip          = "wireguard-exported.zip"
	MaxWireguardRecords       = 10
)

var (
	WithExportWireguardZip = defaultExportWireguardZip
	ZipOutput              = defaultOutputZip
)

func ExportWireguardZip(datas []CloudflareIPData) {
	var fp *os.File
	var err any
	if _, err := os.Stat(ZipOutput); nil != err {
		fp, err = os.Create(ZipOutput)
	} else {
		fp, err = os.OpenFile(ZipOutput, os.O_WRONLY, os.ModePerm)
	}

	defer fp.Close()
	if nil != err {
		panic(err)
	}
	z := zip.NewWriter(fp)
	defer z.Close()
	sample, err := ini.Load("conf/wireguard.conf")
	if nil != err {
		panic(err)
	}
	count := 0
	tmp, err := os.MkdirTemp("", "*")
	for _, data := range datas {
		if count == MaxWireguardRecords {
			break
		}
		name := fmt.Sprintf("wg-config%d-%s.conf", count+1, data.getCountry())
		wr, err := z.CreateHeader(&zip.FileHeader{
			Name:   name,
			Method: zip.Deflate,
		})
		if nil != err {
			panic(err)
		}
		section_peer, err := sample.GetSection("Peer")
		if nil != err {
			panic(err)
		}

		if section_peer.HasKey("Endpoint") {
			section_peer.DeleteKey("Endpoint")
		}
		section_peer.NewKey("Endpoint", data.IP.String())
		tmp_file, err := os.CreateTemp(tmp, "*")
		if err != nil {
			panic(err)
		}
		sample.WriteTo(tmp_file)
		tmp_file.Seek(0, 0)
		io.Copy(wr, tmp_file)
		tmp_file.Close()
		os.Remove(tmp_file.Name())
		count++
	}
	os.Remove(tmp)
}
