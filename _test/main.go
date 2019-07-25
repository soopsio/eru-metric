package main

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"log"
)

func main() {
	c, err := client.NewClientWithOpts(client.FromEnv,client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalln(err)
	}

	log.Println(c.ClientVersion())
	ctx:=context.TODO()
	containers,err:=c.ContainerList(ctx,types.ContainerListOptions{})
	if err != nil {
		log.Fatalln(err)
	}
	for _,d:=range containers{
		log.Printf("%+v",d)
	}
}
