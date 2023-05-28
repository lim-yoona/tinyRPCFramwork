package code

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"io"
	"tinyRPCFramwork/irpc"
)

type GobCode struct {
	// 得到的链接实例
	conn io.ReadWriteCloser
	// bufio包提供了缓冲输入输出功能，可以用于
	// 高效地读写数据流
	// 通过使用bufio包，可以避免频繁系统调用，从而提高生活性能
	buf *bufio.Writer
	dec *gob.Decoder
	enc *gob.Encoder
}

var _ irpc.ICode = (*GobCode)(nil)

func NewGobCode(conn io.ReadWriteCloser) irpc.ICode {
	buf := bufio.NewWriter(conn)
	return &GobCode{
		conn: conn,
		buf:  buf,
		dec:  gob.NewDecoder(conn),
		enc:  gob.NewEncoder(buf),
	}
}
func (gc *GobCode) ReadHeader(header *irpc.Header) error {
	return gc.dec.Decode(header)
}
func (gc *GobCode) ReadBody(body interface{}) error {
	return gc.dec.Decode(body)
}
func (gc *GobCode) Write(header *irpc.Header, body interface{}) (err error) {
	defer func() {
		_ = gc.buf.Flush()
		if err != nil {
			gc.Close()
		}
	}()
	err = gc.enc.Encode(header)
	if err != nil {
		fmt.Println("Header Writer encode err: ", err)
		return err
	}
	err = gc.enc.Encode(body)
	if err != nil {
		fmt.Println("Body Writer encode err: ", err)
		return err
	}
	return nil
}
func (gc *GobCode) Close() error {
	return gc.conn.Close()
}
func init() {
	irpc.NewCodeFuncMap = make(map[irpc.Type]irpc.NewCodeFunc)
	irpc.NewCodeFuncMap[irpc.GobType] = NewGobCode
}
