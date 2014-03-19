package main

import (
	"bytes"
	"encoding/gob"
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

var RedisPool *redis.Pool

func (d *Dispatcher) Dispatch(req *proto.DispatchRequest, resp *proto.DispatchResponse) error {
	d.count++
	log.Println("count=", d.count, "req: ", req)
	resp.SessionID = fmt.Sprintf("%d", time.Now().UnixNano())

	sessionID := resp.SessionID

	conn := RedisPool.Get()
	defer conn.Close()

	args := make([]interface{}, 1+2*len(req.Hosts))
	args[0] = sessionID
	for i, h := range req.Hosts {
		args[1+2*i] = h
		args[1+2*i+1] = ""
	}

	if _, err := conn.Do("HMSET", args...); err != nil {
		err := fmt.Errorf("failed to set redis key %s: %s", resp.SessionID, err)
		return err
	}

	// command output expires in redis 60 seconds after command timeout
	expire := time.Duration(req.Timeout+60) * time.Second
	if _, err := conn.Do("EXPIRE", sessionID, expire.Seconds()); err != nil {
		err := fmt.Errorf("failed to set redis key %s: %s", resp.SessionID, err)
		return err
	}

	for _, host := range req.Hosts {

		go func(host string) {

			client, dialErr := rpc.DialHTTP("tcp", host)
			if dialErr != nil {
				log.Println("error dialing agent:", dialErr)

				req := proto.OutputRequest{
					SessionID: sessionID,
					Host:      host,
					Err:       dialErr.Error(),
				}

				// error ignored
				writeOutputToRedis(&req)

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

				req := proto.OutputRequest{
					SessionID: sessionID,
					Host:      host,
					Err:       runErr.Error(),
				}

				// error ignored
				writeOutputToRedis(&req)

				return
			}
		}(host)
	}

	return nil
}

func (d *Dispatcher) CommandOutput(req *proto.OutputRequest, resp *proto.Status) error {
	log.Println("output:", req)

	// we just wrap this one call at the moment
	err := writeOutputToRedis(req)

	if err != nil {
		resp.Code = http.StatusInternalServerError
	} else {
		resp.Code = http.StatusOK
	}
	return nil
}

func writeOutputToRedis(req *proto.OutputRequest) error {
	conn := RedisPool.Get()
	defer conn.Close()

	// FIXME(dgryski): check if the output is "too old" and discard?

	b := &bytes.Buffer{}
	e := gob.NewEncoder(b)
	err := e.Encode(req)
	if err != nil {
		log.Println(err)
		return err
	}

	if _, err := conn.Do("HSET", req.SessionID, req.Host, b.Bytes()); err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func (d *Dispatcher) Result(req *proto.ResultRequest, resp *proto.ResultResponse) error {

	log.Println("result:", req)

	conn := RedisPool.Get()
	defer conn.Close()

	values, err := redis.Values(conn.Do("HGETALL", req.SessionID))
	if err != nil {
		return err
	}

	// host => output
	m := make(map[string]proto.OutputRequest)
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

		if len(v) > 0 {
			var out proto.OutputRequest
			d := gob.NewDecoder(bytes.NewReader(v))
			err := d.Decode(&out)
			if err != nil {
				log.Println("error decoding buffer: ", err)
				return errors.New("failed to decode output: " + err.Error())
			}

			m[string(k)] = out
		}
	}

	resp.SessionID = req.SessionID
	resp.Output = m

	return nil
}

func main() {

	port := flag.Int("port", 8080, "listen port")
	redisServer := flag.String("redis", "localhost:6379", "redis connect string")

	flag.Parse()

	RedisPool = redis.NewPool(
		func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", *redisServer)
			if err != nil {
				return nil, err
			}
			return c, err
		},
		3,
	)

	dispatch := new(Dispatcher)

	rpc.Register(dispatch)

	rpc.HandleHTTP()
	l, e := net.Listen("tcp", ":"+strconv.Itoa(*port))
	if e != nil {
		log.Fatal("listen error:", e)
	}
	http.Serve(l, nil)

}
