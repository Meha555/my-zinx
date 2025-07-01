package session

import (
	"errors"
	"fmt"
	"io"
	"pulse/core/message"
	"pulse/server/common"
	"pulse/server/job"

	"net"
	utils "pulse/utils"
	"sync/atomic"

	"pulse/logging"

	"github.com/google/uuid"
)

var logger = logging.NewStdLogger(logging.LevelInfo, "session", "[%t] [%c %l] [%f:%C:%L:%g] %m", false)

// Session
// 将裸的TCP socket包装，将具体的业务与连接绑定
type Session struct {
	// 当前连接的socket TCP套接字
	conn *net.TCPConn
	// 当前连接的ID 也可以称作为SessionID，ID全局唯一
	sessionID uuid.UUID
	// 当前连接的关闭状态
	isClosed atomic.Bool
	// 保活心跳次数
	heartbeat uint

	// 工作协程池
	workerPool *job.WorkerPool

	// 用于读写协程(Reader/Writer)之间的通信（用于实现读写业务分离）
	msgCh chan []byte
	// 通知该连接已经停止
	exitCh chan struct{}

	hookStub hooks
}

func NewSession(conn *net.TCPConn, workerPool *job.WorkerPool, hookOpts ...hookOpt) *Session {
	c := &Session{
		conn:       conn,
		sessionID:  uuid.New(),
		isClosed:   atomic.Bool{},
		heartbeat:  0,
		workerPool: workerPool,
		msgCh:      make(chan []byte, utils.Conf.Server.MaxMsgQueueSize), // 这里设置缓冲区大小为10，允许读写协程的处理速率有一定的差异
		exitCh:     make(chan struct{}, 1),                               // 这里设置为 1，确保至少有一个缓冲区，防止写入时没人读导致阻塞，或者反之
		hookStub: hooks{
			OnOpen:     noOp,
			OnClose:    noOp,
			BeforeSend: noOp,
			BeforeRecv: noOp,
			AfterSend:  noOp,
			AfterRecv:  noOp,
		},
	}

	for _, opt := range hookOpts {
		opt(c)
	}

	return c
}

func (c *Session) Open() error {
	// 启动IO协程负责该连接的读写操作
	go c.Reader()
	go c.Writer()

	c.hookStub.OnOpen(c)

	// 等待 Stop() 方法通知退出
	for range c.exitCh {
		return nil
	}
	return nil
}

func (c *Session) Close() {
	if c.isClosed.Load() {
		return
	}
	c.isClosed.Store(true)

	c.hookStub.OnClose(c)

	c.conn.Close()
	c.exitCh <- struct{}{} // 通知 Open() 方法退出
	close(c.msgCh)
	close(c.exitCh)
}

func (c *Session) ID() uuid.UUID {
	return c.sessionID
}

func (c *Session) Conn() net.Conn {
	return c.conn
}

func (c *Session) UpdateHeartBeat() {
	c.heartbeat = 0
}

func (c *Session) HeartBeat() uint {
	return c.heartbeat
}

func (c *Session) Send(data []byte) (int, error) {
	if c.isClosed.Load() {
		return 0, errors.New("connection is closed")
	}
	return c.conn.Write(data)
}

func (c *Session) Recv(data []byte) (int, error) {
	if c.isClosed.Load() {
		return 0, errors.New("connection is closed")
	}
	return c.conn.Read(data)
}

func (c *Session) SendMsg(msg message.IPacket) error {
	if c.isClosed.Load() {
		return errors.New("connection is closed")
	}
	c.hookStub.BeforeSend(c)
	defer c.hookStub.AfterSend(c)
	data, err := message.Marshal(msg)
	if err != nil {
		return err
	}
	// return c.conn.Write(data)
	// 提交给让Writer协程异步发送，这样不会因为底层TCP发送缓冲区满而导致这里阻塞
	// 如果发送有错误，则由Writer协程处理，这里直接返回
	c.msgCh <- data
	return nil
}

// NOTE 这种接口作为传出参数，不用指针可以实现传出修改
func (c *Session) RecvMsg(msg message.IPacket) error {
	if c.isClosed.Load() {
		return errors.New("connection is closed")
	}
	c.hookStub.BeforeRecv(c)
	defer c.hookStub.AfterRecv(c)
	headerData := make([]byte, msg.HeaderLen())
	if _, err := io.ReadFull(c.conn, headerData); err != nil {
		return fmt.Errorf("read header error: %w", err)
	}
	if err := message.Unmarshal(headerData, msg, false); err != nil {
		return fmt.Errorf("unmarshal header err: %w", err)
	}
	// 读取负载
	if msg.BodyLen() <= 0 {
		return nil
	}
	bodyData := make([]byte, msg.BodyLen())
	if _, err := io.ReadFull(c.conn, bodyData); err != nil {
		return fmt.Errorf("read body error: %w", err)
	}
	if err := message.UmarshalBodyOnly(bodyData, int(msg.BodyLen()), msg); err != nil {
		return fmt.Errorf("unmarshal body error: %w", err)
	}
	return nil
}

func (c *Session) ExitChan() <-chan struct{} {
	return c.exitCh
}

// 确保 Connection 实现 IConenction 方法
var _ common.ISession = (*Session)(nil)

// Reader 是用于读取客户端数据的 Goroutine
// 会需要与主协程通过chan通信
func (c *Session) Reader() {
	logger.Debug("Reader Goroutine is running")
	defer logger.Debugf(c.Conn().RemoteAddr().String(), " Reader Goroutine exit!")
	defer c.Close() // 确保连接能被关闭

	for {
		msg := &message.SeqedTLVMsg{}
		if err := c.RecvMsg(msg); err != nil {
			logger.Errorf("RecvMsg error: %v", err)
			c.Close()
			return
		}
		// 封装请求数据
		req := GetRequest(c, msg)
		// 提交给协程池来处理业务
		c.workerPool.Post(req)
	}
}

// Writer 是用于向客户端发送数据的 Goroutine
// 会需要与主协程通过chan通信
func (c *Session) Writer() {
	logger.Debug("Writer Goroutine is running")
	defer logger.Debugf(c.Conn().RemoteAddr().String(), " Writer Goroutine exit!")
	for {
		select {
		case data := <-c.msgCh: // 从msgCh中读取数据
			if _, err := c.Send(data); err != nil {
				logger.Errorf("Send error: %v", err)
				continue
			}
		case <-c.exitCh: // 响应退出信号
			return
		}
	}
}
