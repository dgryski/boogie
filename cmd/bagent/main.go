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
	"syscall"
	"time"

	"github.com/dgryski/boogie/proto"
)

type Agent struct{}

func (agent *Agent) RunCommand(req *proto.RunRequest, resp *proto.Status) error {
	log.Println("req: ", req)

	go func(req *proto.RunRequest) {

		time.Sleep(Delay) // for testing

		done := make(chan error)

		cmd := exec.Command("/bin/bash", "-c", req.Command)

		var sout bytes.Buffer
		var serr bytes.Buffer
		cmd.Stdout = &sout
		cmd.Stderr = &serr

		go func() {
			err := cmd.Run()
			done <- err
		}()

		timeout := time.After(time.Duration(req.Timeout) * time.Second)

		out := proto.OutputRequest{
			Host:      Name,
			SessionID: req.SessionID,
		}

		select {
		case err := <-done:
			if err != nil {
				out.Err = err.Error()
			}

			out.Stdout = sout.Bytes()
			out.Stderr = serr.Bytes()
			if cmd.ProcessState != nil {
				out.ExitCode = cmd.ProcessState.Sys().(syscall.WaitStatus).ExitStatus()
			}
		case <-timeout:
			out.Err = "(timeout)"
		}

		m, err := rpc.Dial("tcp", req.ResponseHost)
		if err != nil {
			log.Println("rpc.Dial:", err)
			return
		}

		var resp proto.Status
		m.Call("Dispatcher.CommandOutput", &out, &resp)

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

	l, e := net.Listen("tcp", ":"+strconv.Itoa(*port))
	if e != nil {
		log.Fatal("listen error:", e)
	}
	rpc.Accept(l)

}
