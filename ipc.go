package entityipc

import (
	"bufio"
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

type message struct {
	Id   uint32 `json:"id"`
	Data []byte `json:"data"`
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
		msgTxt, err := r.ReadBytes('\000')

		if err != nil {
			return err
		}

		// Remove null char
		msgTxt = msgTxt[:len(msgTxt)-1]

		var msg message
		if err := json.Unmarshal(msgTxt, &msg); err != nil {
			return err
		}

		ipc.mutex.RLock()
		c, ok := ipc.messageMap[msg.Id]
		ipc.mutex.RUnlock()

		if ok {
			c <- msg.Data
		} else {
			if ipc.targetType == nil {
				ipc.targetType = make(map[string]interface{})
			}

			targetType := reflect.New(reflect.TypeOf(ipc.targetType)).Elem()
			json.Unmarshal(msg.Data, targetType.Addr().Interface())

			responseMsg := message{
				Id: msg.Id,
			}
			res, err := ipc.handler(targetType.Interface())

			if err != nil {
				responseMsg.Data = errorToJson(err)
			} else {
				responseMsg.Data, _ = json.Marshal(res)
			}

			data, _ := json.Marshal(responseMsg)
			ipc.rw.Write(append(data, '\000'))
		}
	}
}

func (ipc *IPC) Handle(targetType interface{}, fn HandlerFunc) {
	ipc.handler = fn
	ipc.targetType = targetType
}

func (ipc *IPC) Send(v interface{}, response interface{}) error {
	c := make(chan []byte)
	msg := message{
		Id: rand.Uint32(),
	}

	jsonData, err := json.Marshal(v)

	if err != nil {
		return err
	}

	msg.Data = jsonData
	msgJson, _ := json.Marshal(msg)
	if _, err := ipc.rw.Write(append(msgJson, '\000')); err != nil {
		return err
	}

	ipc.mutex.Lock()
	ipc.messageMap[msg.Id] = c
	ipc.mutex.Unlock()
	resJson := <-c

	ipc.mutex.Lock()
	delete(ipc.messageMap, msg.Id)
	ipc.mutex.Unlock()

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
