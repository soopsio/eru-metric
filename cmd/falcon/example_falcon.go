package main

import (
	"flag"
	"fmt"
	"github.com/docker/docker/api/types"
	"os"
	"time"

	"golang.org/x/net/context"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker/client"
	"github.com/soopsio/eru-metric/falcon"
	"github.com/soopsio/eru-metric/metric"
)

func main() {
	var dockerAddr string
	var transferAddr string
	var certDir string
	var debug bool
	flag.BoolVar(&debug, "DEBUG", false, "enable debug")
	flag.StringVar(&dockerAddr, "d", "tcp://0.0.0.0:2376", "docker daemon addr")
	flag.StringVar(&transferAddr, "t", "0.0.0.0:8433", "transfer addr")
	flag.StringVar(&certDir, "c", "/root/.docker", "cert files dir")
	flag.Parse()
	//if flag.NArg() < 1 {
	//	fmt.Println("need at least one container id")
	//	return
	//}
	if debug {
		log.SetLevel(log.DebugLevel)
	}

	log.Info(dockerAddr)
	cli, err := client.NewClientWithOpts(client.WithHost(dockerAddr), client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalln(err)
	}

	metric.SetGlobalSetting(cli, 2, 3, "vnbe", "eth0")
	falconClient := falcon.CreateFalconClient(transferAddr, 5*time.Millisecond)
	ctx := context.Background()

	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		log.Fatalln(err)
	}
	for _, d := range containers {
		if c, err := cli.ContainerInspect(ctx, d.ID); err != nil {
			fmt.Println(d.ID, d.Names, err)
			continue
		} else {
			go start_watcher(falconClient,c)
		}
	}
	//for i := 0; i < flag.NArg(); i++ {
	//	if c, err := cli.ContainerInspect(ctx, flag.Arg(i)); err != nil {
	//		fmt.Println(flag.Arg(i), err)
	//		continue
	//	} else {
	//		go start_watcher(client, c.ID, c.State.Pid)
	//	}
	//}
	select {}
}

func start_watcher(client metric.Remote, container types.ContainerJSON) {
	hostname,_:=os.Hostname()
	serv := metric.CreateMetric(time.Duration(5)*time.Second, client, "", fmt.Sprintf("container—_%s_%s", hostname,container.Name))
	defer serv.Client.Close()
	if err := serv.InitMetric(container.ID, container.State.Pid); err != nil {
		fmt.Println("failed", err)
		return
	}

	t := time.NewTicker(serv.Step)
	defer t.Stop()
	fmt.Println("begin watch", container.ID, container.Name)
	for {
		select {
		case now := <-t.C:
			go func() {
				if info, err := serv.UpdateStats(container.ID); err == nil {
					fmt.Println(info)
					rate := serv.CalcRate(info, now)
					serv.SaveLast(info)
					// for safe
					fmt.Println(rate)
					go serv.Send(rate)
				}
			}()
		case <-serv.Stop:
			return
		}
	}
}
