package entityipc

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
)

type HandlerFunc func(v interface{}) (interface{}, error)

type IPC struct {
	conn       net.Conn
	messageMap map[uint32]chan []byte
	handler    HandlerFunc
	targetType interface{}
}

func (ipc *IPC) Start(conn net.Conn) error {
	ipc.messageMap = make(map[uint32]chan []byte)
	ipc.conn = conn

	if ipc.handler == nil {
		ipc.handler = func(v interface{}) (interface{}, error) {
			return nil, fmt.Errorf("no handler registered")
		}
	}

	r := bufio.NewReader(conn)

	for {
		msg, err := r.ReadBytes('\000')

		if err != nil {
			return err
		}

		msg = msg[:1]

		if len(msg) < 4 {
			continue
		}
		id := binary.BigEndian.Uint32(msg[:4])

		if c, ok := ipc.messageMap[id]; ok {
			c <- msg[4:]
		} else {
			targetType := ipc.targetType

			if ipc.targetType != nil {
				json.Unmarshal(msg[4:], &targetType)
			}

			var data []byte
			binary.BigEndian.PutUint32(data, id)
			res, err := ipc.handler(targetType)

			if err != nil {
				data = append(data, errorToJson(err)...)
			} else {
				jsonData, _ := json.Marshal(res)
				data = append(data, jsonData...)
			}

			ipc.conn.Write(append(data, '\000'))
		}
	}
}

func (ipc *IPC) Handle(targetType interface{}, fn HandlerFunc) {
	ipc.handler = fn
	ipc.targetType = targetType
}

func (ipc *IPC) Send(v interface{}, response interface{}) error {
	id := rand.Uint32()
	c := make(chan []byte)
	var data []byte

	binary.BigEndian.PutUint32(data, id)
	jsonData, err := json.Marshal(v)

	if err != nil {
		return err
	}

	data = append(data, jsonData...)
	data = append(data, '\000')
	if _, err := ipc.conn.Write(data); err != nil {
		return err
	}

	ipc.messageMap[id] = c
	resJson := <-c

	return json.Unmarshal(resJson, response)
}

func errorToJson(err error) []byte {
	errStruct := struct {
		Error   error
		Message string
	}{err, err.Error()}

	b, _ := json.Marshal(errStruct)

	return b
}
