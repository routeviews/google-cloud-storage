package main

import (
	"flag"
	"os"

	"github.com/golang/glog"
	converter "github.com/routeviews/google-cloud-storage/pkg/mrt_converter"
)

var (
	collector = flag.String("collector", "", "Collector name of this archive.")
	archive   = flag.String("archive", "", "Path to the bz2 MRT archive.")
	output    = flag.String("output", "", "Output path of the converted archive.")
)

func main() {
	flag.Parse()
	src, err := os.Open(*archive)
	if err != nil {
		glog.Exit(err)
	}
	defer src.Close()
	dst, err := os.Create(*output)
	if err != nil {
		glog.Exit(err)
	}
	defer dst.Close()

	converter.Convert(*collector, src, dst)
}
