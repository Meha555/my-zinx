package session

import "pulse/server/common"

type hooks struct {
	OnOpen     hook
	OnClose    hook
	BeforeSend hook
	BeforeRecv hook
	AfterSend  hook
	AfterRecv  hook
}

type hook func(common.ISession)
type hookOpt func(c *Session)

// 定义一个空函数
var noOp hook = func(common.ISession) {}

func onOpen(f hook) hookOpt {
	return func(c *Session) {
		c.hookStub.OnOpen = f
	}
}

func onClose(f hook) hookOpt {
	return func(c *Session) {
		c.hookStub.OnClose = f
	}
}

func beforeSend(f hook) hookOpt {
	return func(c *Session) {
		c.hookStub.BeforeSend = f
	}
}

func beforeRecv(f hook) hookOpt {
	return func(c *Session) {
		c.hookStub.BeforeRecv = f
	}
}

func afterSend(f hook) hookOpt {
	return func(c *Session) {
		c.hookStub.AfterSend = f
	}
}

func afterRecv(f hook) hookOpt {
	return func(c *Session) {
		c.hookStub.AfterRecv = f
	}
}
