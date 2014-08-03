package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"io/ioutil"
	"log"
	"net"
	"net/rpc"
	"strings"
	"time"

	"github.com/dgryski/boogie/proto"
)

func main() {

	cmd := flag.String("c", "ls", "command to run")
	master := flag.String("m", "localhost:8079", "rpc address of master server")
	tlsClientCert := flag.String("cert", "", "TLS client certificate")
	tlsPrivateKey := flag.String("priv", "", "TLS private key")
	hosts := flag.String("h", "", "list of hosts to execute on")
	timeout := flag.Int("t", 10, "execution timeout")

	flag.Parse()

	var rpcConn net.Conn
	var err error
	if *tlsClientCert != "" {
		cert2_b, _ := ioutil.ReadFile(*tlsClientCert)
		priv2_b, _ := ioutil.ReadFile(*tlsPrivateKey)
		priv2, _ := x509.ParsePKCS1PrivateKey(priv2_b)

		cert := tls.Certificate{
			Certificate: [][]byte{cert2_b},
			PrivateKey:  priv2,
		}

		config := tls.Config{Certificates: []tls.Certificate{cert}, InsecureSkipVerify: true}
		rpcConn, err = tls.Dial("tcp", *master, &config)
	} else {
		rpcConn, err = net.Dial("tcp", *master)
	}
	if err != nil {
		log.Fatal("error dialing:", err)
	}
	defer rpcConn.Close()

	client := rpc.NewClient(rpcConn)
	req := proto.DispatchRequest{
		Command: *cmd,
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
