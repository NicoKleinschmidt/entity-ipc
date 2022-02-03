package entityipc

import (
	"bufio"
	"encoding/json"
	"net"
	"time"
)

type JsonIpcConn interface {
	net.Conn

	WriteObject(v interface{}) (int, error)
	ReadObject(v interface{}) (int, error)
}

type jsonIpcConn struct {
	conn net.Conn
}

func Dial(address string) (JsonIpcConn, error) {
	conn, err := net.Dial("unix", address)

	if err != nil {
		return nil, err
	}

	return jsonIpcConn{conn}, nil
}

func (c jsonIpcConn) ReadObject(v interface{}) (int, error) {
	r := bufio.NewReader(c.conn)
	data, err := r.ReadBytes('\000')

	if err != nil {
		return len(data), err
	}

	err = json.Unmarshal(data[:len(data)-1], v)

	return len(data), err
}

func (c jsonIpcConn) WriteObject(v interface{}) (int, error) {
	data, err := json.Marshal(v)

	if err != nil {
		return 0, err
	}

	return c.conn.Write(append(data, '\000'))
}

func (c jsonIpcConn) Read(b []byte) (int, error) {
	return c.conn.Read(b)
}

func (c jsonIpcConn) Write(b []byte) (int, error) {
	return c.conn.Write(b)
}

func (c jsonIpcConn) Close() error {
	return c.conn.Close()
}

func (c jsonIpcConn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c jsonIpcConn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c jsonIpcConn) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

func (c jsonIpcConn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

func (c jsonIpcConn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}

type JsonIpcListener interface {
	net.Listener

	AcceptJson() (JsonIpcConn, error)
}

type jsonIpcListener struct {
	ls net.Listener
}

func Listen(address string) (JsonIpcListener, error) {
	ls, err := net.Listen("unix", address)

	if err != nil {
		return nil, err
	}

	return jsonIpcListener{ls}, nil
}

func (ipcLs jsonIpcListener) AcceptJson() (JsonIpcConn, error) {
	conn, err := ipcLs.ls.Accept()

	if err != nil {
		return nil, err
	}

	return jsonIpcConn{
		conn: conn,
	}, nil
}

func (ipcLs jsonIpcListener) Accept() (net.Conn, error) {
	return ipcLs.AcceptJson()
}

func (ipcLs jsonIpcListener) Close() error {
	return ipcLs.ls.Close()
}

func (ipcLs jsonIpcListener) Addr() net.Addr {
	return ipcLs.ls.Addr()
}
