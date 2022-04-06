package entityipc

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"reflect"
	"sync"
)

type HandlerFunc func(v interface{}) (interface{}, error)

type IPC struct {
	rw         io.ReadWriter
	messageMap map[uint32]chan []byte
	handler    HandlerFunc
	targetType interface{}
	mutex      *sync.RWMutex
}

func (ipc *IPC) Start(rw io.ReadWriter) error {
	ipc.messageMap = make(map[uint32]chan []byte)
	ipc.rw = rw
	ipc.mutex = &sync.RWMutex{}

	if ipc.handler == nil {
		ipc.handler = func(v interface{}) (interface{}, error) {
			return nil, fmt.Errorf("no handler registered")
		}
	}

	r := bufio.NewReader(rw)

	for {
		msg, err := r.ReadBytes('\000')

		if err != nil {
			return err
		}

		msg = msg[:len(msg)-1]

		if len(msg) < 4 {
			continue
		}
		id := binary.BigEndian.Uint32(msg[:4])

		ipc.mutex.RLock()
		c, ok := ipc.messageMap[id]
		ipc.mutex.RUnlock()

		if ok {
			c <- msg[4:]
		} else {
			if ipc.targetType == nil {
				ipc.targetType = make(map[string]interface{})
			}

			targetType := reflect.New(reflect.TypeOf(ipc.targetType)).Elem()
			json.Unmarshal(msg[4:], targetType.Addr().Interface())

			data := make([]byte, 4)
			binary.BigEndian.PutUint32(data, id)
			res, err := ipc.handler(targetType.Interface())

			if err != nil {
				data = append(data, errorToJson(err)...)
			} else {
				jsonData, _ := json.Marshal(res)
				data = append(data, jsonData...)
			}

			ipc.rw.Write(append(data, '\000'))
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
	data := make([]byte, 4)

	binary.BigEndian.PutUint32(data, id)
	jsonData, err := json.Marshal(v)

	if err != nil {
		return err
	}

	data = append(data, jsonData...)
	data = append(data, '\000')
	if _, err := ipc.rw.Write(data); err != nil {
		return err
	}

	ipc.mutex.Lock()
	ipc.messageMap[id] = c
	ipc.mutex.Unlock()
	resJson := <-c

	delete(ipc.messageMap, id)
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
