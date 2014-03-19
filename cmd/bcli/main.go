package main

import (
	"flag"
	"log"
	"net/rpc"
	"strings"
	"time"

	"github.com/dgryski/boogie/proto"
)

func main() {

	master := flag.String("m", "localhost:8080", "address of master server")
	hosts := flag.String("h", "", "list of hosts to execute on")
	timeout := flag.Int("t", 10, "execution timeout")

	flag.Parse()

	client, err := rpc.DialHTTP("tcp", *master)
	if err != nil {
		log.Fatal("error dialing:", err)
	}

	req := proto.DispatchRequest{
		Command: flag.Args(),
		Hosts:   strings.Split(*hosts, ","),
		Timeout: *timeout,
	}

	var resp proto.DispatchResponse

	log.Println("req=", req)

	err = client.Call("Dispatcher.Dispatch", &req, &resp)

	if err != nil {
		log.Fatal("error calling:", err)
	}

	log.Println("response: ", resp)

	resultReq := proto.ResultRequest{
		SessionID: resp.SessionID,
	}

	seen := make(map[string]bool)

	for i := 0; i < 10 && len(seen) < len(req.Hosts); i++ {

		var resultResp proto.ResultResponse
		err = client.Call("Dispatcher.Result", &resultReq, &resultResp)
		if err != nil {
			log.Fatal("err calling result: ", err)
		}

		time.Sleep(1 * time.Second)
		for host, output := range resultResp.Output {
			if !seen[host] {
				log.Printf("host: %s\nstderr: %s\nstdout: %s\nerr: %s\nexit: %d\n", host, string(output.Stderr), string(output.Stdout), output.Err, output.ExitCode)
				seen[host] = true
			}
		}
	}
}
