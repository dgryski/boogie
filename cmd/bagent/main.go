package main

import (
	"bytes"
	"flag"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os/exec"
	"strconv"
	"time"

	"github.com/dgryski/boogie/proto"
)

type Agent struct{}

func (agent *Agent) RunCommand(req *proto.RunRequest, resp *proto.Status) error {
	log.Println("req: ", req)

	go func(req *proto.RunRequest) {

		time.Sleep(Delay) // for testing

		done := make(chan error)

		cmd := exec.Command(req.Command[0], req.Command[1:]...)

		var sout bytes.Buffer
		cmd.Stdout = &sout

		go func() {
			err := cmd.Run()
			done <- err
		}()

		timeout := time.After(time.Duration(req.Timeout) * time.Second)

		var out []byte
		select {
		case err := <-done:
			if err != nil {
				out = []byte(err.Error())
				break
			}

			out = sout.Bytes()
			if len(out) == 0 {
				out = []byte("(none)")
			}
		case <-timeout:
			out = []byte("(timeout)")
		}

		m, err := rpc.DialHTTP("tcp", req.ResponseHost)
		if err != nil {
			log.Println("rpc.DialHTTP:", err)
			return
		}

		var resp proto.Status
		m.Call("Dispatcher.CommandOutput", &proto.OutputRequest{
			Host:      Name,
			SessionID: req.SessionID,
			Output:    out,
		}, &resp)

		if resp.Code != http.StatusOK {
			log.Println("error sending output")
			return
		}
	}(req)

	resp.Code = http.StatusOK
	return nil
}

var Name string

var Delay time.Duration

func main() {

	flag.StringVar(&Name, "name", "localhost:8081", "name of this node")

	port := flag.Int("port", 8081, "listen port")
	flag.DurationVar(&Delay, "delay", 1*time.Second, "delay before running commands")

	flag.Parse()

	rpc.Register(new(Agent))

	rpc.HandleHTTP()
	l, e := net.Listen("tcp", ":"+strconv.Itoa(*port))
	if e != nil {
		log.Fatal("listen error:", e)
	}
	http.Serve(l, nil)

}
