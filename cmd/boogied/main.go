package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"reflect"
	"strconv"
	"time"

	"github.com/dgryski/boogie/proto"
	"github.com/garyburd/redigo/redis"
)

type Dispatcher struct {
	count int
}

func (d *Dispatcher) Dispatch(req *proto.DispatchRequest, resp *proto.DispatchResponse) error {
	d.count++
	log.Println("count=", d.count, "req: ", req)
	resp.SessionID = fmt.Sprintf("%d", time.Now().UnixNano())

	conn, err := redis.Dial("tcp", ":6379")
	if err != nil {
		log.Println("redis.Dial():", err)
		return errors.New("unable to contact redis: " + err.Error())
	}
	defer conn.Close()

	sessionID := resp.SessionID

	for _, host := range req.Hosts {

		// add this host to our redis store
		if _, err := conn.Do("HSET", sessionID, host, nil); err != nil {
			log.Printf("error setting redis key %s/%s: %s", resp.SessionID, host, err)
		}

		go func(host string) {

			client, dialErr := rpc.DialHTTP("tcp", host)
			if dialErr != nil {
				log.Println("error dialing agent:", dialErr)

				conn, err := redis.Dial("tcp", ":6379")
				if err != nil {
					log.Println("redis.Dial():", err)
					return
				}
				defer conn.Close()

				// add this host's error to our redis store
				if _, err := conn.Do("HSET", sessionID, host, dialErr); err != nil {
					log.Printf("error setting redis key %s/%s: %s", resp.SessionID, host, err)
				}

				return
			}

			req := proto.RunRequest{
				ResponseHost: "localhost:8080",
				Command:      req.Command,
				SessionID:    sessionID,
				Timeout:      req.Timeout,
			}

			var resp proto.Status

			log.Println("req=", req)

			runErr := client.Call("Agent.RunCommand", &req, &resp)

			if runErr != nil {

				log.Println("error calling Agent.RunCommand:", runErr)

				conn, err := redis.Dial("tcp", ":6379")
				if err != nil {
					log.Println("redis.Dial():", err)
					return
				}
				defer conn.Close()

				// add this host's error to our redis store
				if _, err := conn.Do("HSET", sessionID, host, runErr); err != nil {
					log.Printf("error setting redis key %s/%s: %s", sessionID, host, err)
				}
				return
			}
		}(host)
	}

	return nil
}

func (d *Dispatcher) CommandOutput(req *proto.OutputRequest, resp *proto.Status) error {
	log.Println("output:", req)

	conn, err := redis.Dial("tcp", ":6379")
	if err != nil {
		log.Println("redis.Dial():", err)
		return errors.New("unable to contact redis: " + err.Error())
	}
	defer conn.Close()

	// FIXME(dgryski): check if the output is "too old" and discard?

	if _, err := conn.Do("HSET", req.SessionID, req.Host, req.Output); err != nil {
		log.Println(err)
	}

	resp.Code = http.StatusOK
	return nil
}

func (d *Dispatcher) Result(req *proto.ResultRequest, resp *proto.ResultResponse) error {

	log.Println("result:", req)

	conn, err := redis.Dial("tcp", ":6379")
	if err != nil {
		log.Println("redis.Dial():", err)
		return errors.New("unable to contact redis: " + err.Error())
	}
	defer conn.Close()

	values, err := redis.Values(conn.Do("HGETALL", req.SessionID))
	if err != nil {
		return err
	}

	// host => output
	m := make(map[string]string)
	for i := 0; i < len(values); i += 2 {
		var k []uint8
		var kok bool
		if k, kok = values[i].([]uint8); !kok {
			return errors.New("bad type for key: " + reflect.TypeOf(values[i]).String())
		}

		var v []uint8
		var vok bool
		if v, vok = values[i+1].([]uint8); !vok {
			return errors.New("bad type for value: " + reflect.TypeOf(values[i+1]).String())
		}

		log.Println(v)

		m[string(k)] = string(v)
	}

	resp.SessionID = req.SessionID
	resp.Output = m

	return nil
}

func main() {

	port := flag.Int("port", 8080, "listen port")

	flag.Parse()

	dispatch := new(Dispatcher)

	rpc.Register(dispatch)

	rpc.HandleHTTP()
	l, e := net.Listen("tcp", ":"+strconv.Itoa(*port))
	if e != nil {
		log.Fatal("listen error:", e)
	}
	http.Serve(l, nil)

}
