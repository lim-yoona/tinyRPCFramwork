package encode

import (
	"bufio"
	"encoding/gob"
	"io"
	"log"
)

// 创建一个Gob对象，实现Encoder接口
type GobEncoder struct {
	// conn是通过TCP或者UNIX建立socket时得到的链接实例
	conn io.ReadWriteCloser
	// 创建的带缓冲的writer，防止频繁阻塞，提高性能
	buf *bufio.Writer
	// gob的编码器
	enc *gob.Encoder
	// gob的解码器
	dec *gob.Decoder
}

var _ Encoder = (*GobEncoder)(nil)

// 定义构造函数
func NewGobEncoder(conn io.ReadWriteCloser) Encoder {
	buf := bufio.NewWriter(conn)
	return &GobEncoder{
		conn: conn,
		buf:  buf,
		enc:  gob.NewEncoder(buf),
		dec:  gob.NewDecoder(conn),
	}
}

// 实现Encoder接口的方法
func (self *GobEncoder) ReadHeader(h *Header) error {
	return self.dec.Decode(h)
}
func (self *GobEncoder) ReadBody(b interface{}) error {
	return self.dec.Decode(b)
}
func (self *GobEncoder) Write(h *Header, b interface{}) (err error) {
	defer func() {
		_ = self.buf.Flush()
		if err != nil {
			_ = self.Close()
		}
	}()
	if err := self.enc.Encode(h); err != nil {
		log.Println("Encoder: encode Header err", err)
		return err
	}
	if err := self.enc.Encode(b); err != nil {
		log.Println("Encoder: encode Body err", err)
		return err
	}
	return nil
}

func (self *GobEncoder) Close() error {
	return self.conn.Close()
}
