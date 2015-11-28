package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/therealbill/libredis/client"
	"github.com/therealbill/libredis/structures"
)

var (
	app          *cli.App
	slaves       []string
	masterconn   *client.Redis
	sentinelconn *client.Redis
	primarypool  []structures.SlaveInfo
// Version represents the released version of the software 
	Version      string
)

func getSentinelConnection(c *cli.Context) (err error) {
	log.Print("Getting Sentinel connection")
	sentinelconn, err = client.DialAddress(c.GlobalString("sentinel"))
	return err
}

func getMaster(c *cli.Context) (structures.MasterAddress, error) {
	log.Printf("Getting master connection")
	return sentinelconn.SentinelGetMaster(c.GlobalString("podname"))
}

func main() {
	app = cli.NewApp()
	app.Name = "daisy"
	app.Usage = "Create and alter a slavepool in a chained replication configuration"
	app.Version = Version
	app.EnableBashCompletion = true
	author := cli.Author{Name: "Bill Anderson", Email: "therealbill@me.com"}
	app.Authors = append(app.Authors, author)
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "podname, n",
			Usage: "The name of the pod",
		},
		cli.StringFlag{
			Name:   "sentinel, s",
			Value:  "localhost:26379",
			Usage:  "Address of the sentinel ",
			EnvVar: "SentinelAddress",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:  "slavepool",
			Usage: "Actions taken on the secondary slave pool.",
			Subcommands: []cli.Command{
				{
					Name:   "create",
					Usage:  "Create a read-only, non-promotable slave pool.",
					Action: createSlavePool,
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "slaves",
							Usage: "Comma separated list of IP:PORT listing for the slaves to put into the pool",
						},
						cli.StringFlag{
							Name:  "syncpolicy",
							Usage: "The policy to use when enslaving the pool to the master pod.",
							Value: "singleslave",
						},
					},
				},
			},
		},
	}
	app.Run(os.Args)

}

func createSlavePool(c *cli.Context) {
	if c.String("slaves") == "" {
		log.Print("Need a list of existing Redis instances to use as slaves")
		return
	}
	slaves = strings.Split(c.String("slaves"), ",")
	log.Printf("Pod: %s", c.GlobalString("podname"))
	log.Printf("Slave Pool Policy: %s", c.String("syncpolicy"))
	for _, slave := range slaves {
		log.Printf("Secondary Slave: %s\n", slave)
	}
	err := getSentinelConnection(c)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	master, err := getMaster(c)
	if err != nil {
		log.Fatalf("getmaster fail: %v", err)
	}
	log.Printf("Got master : %+v\n", master)
	getPrimarySlavePool(c)
	switch c.String("syncpolicy") {
	case "direct":
		enslaveOneForOne(c)
	case "ring":
		enslaveRing(c)
	case "single":
		enslaveSingleSlave(c)
	}
}

func getPrimarySlavePool(c *cli.Context) {
	log.Print("Pulling primary slavepool")
	var err error
	primarypool, err = sentinelconn.SentinelSlaves(c.GlobalString("podname"))
	if err != nil {
		log.Fatalf("FAIL: %v", err)
	}
	for _, slave := range primarypool {
		log.Printf("Primary Slave Address: %s:%d\n", slave.Host, slave.Port)
	}
}

func enslaveSingleSlave(c *cli.Context) {
	//TODO: make it random?
	target := primarypool[0]
	for _, saddr := range slaves {
		slc, err := client.DialAddress(saddr)
		if err != nil {
			log.Printf("SLAVEPOOL ERROR [%s] %+v\n", saddr, err)
			break
		}
		err = slc.SlaveOf(target.Host, fmt.Sprintf("%d", target.Port))
		if err != nil {
			log.Printf("SLAVEOF ERROR [%s] %+v\n", saddr, err)
		}
		log.Printf("Enslaved %s to %s:%d\n", saddr, target.Host, target.Port)
	}
}

func enslaveOneForOne(c *cli.Context) {
	for x, saddr := range slaves {
		target := primarypool[x]
		slc, err := client.DialAddress(saddr)
		if err != nil {
			log.Printf("SLAVEPOOL ERROR [%s] %+v\n", saddr, err)
			break
		}
		err = slc.SlaveOf(target.Host, fmt.Sprintf("%d", target.Port))
		if err != nil {
			log.Printf("SLAVEOF ERROR [%s] %+v:\n", saddr, err)
		}
		log.Printf("Enslaved %s to %s:%d\n", saddr, target.Host, target.Port)
	}
}

func enslaveRing(c *cli.Context) {
	added := 1
	var target structures.SlaveInfo
	for x, saddr := range slaves {
		index := x % len(primarypool)
		target = primarypool[index]
		slc, err := client.DialAddress(saddr)
		if err != nil {
			log.Printf("SLAVEPOOL ERROR [%s] %+v\n", saddr, err)
			break
		}
		err = slc.SlaveOf(target.Host, fmt.Sprintf("%d", target.Port))
		if err != nil {
			log.Printf("SLAVEOF ERROR [%s] %+v:\n", saddr, err)
		}
		log.Printf("Enslaved %s to %s:%d\n", saddr, target.Host, target.Port)
		added++
	}
}
