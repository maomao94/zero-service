package iec104server

import (
	"github.com/wendy512/go-iecp5/asdu"
	"time"
)

type ServerHandler struct {
	handler CommandHandler
}

func NewServerHandler(commandHandler CommandHandler) *ServerHandler {
	return &ServerHandler{handler: commandHandler}
}

type CommandHandler interface {
	// OnInterrogation 总召唤请求
	OnInterrogation(asdu.Connect, *asdu.ASDU, asdu.QualifierOfInterrogation) error
	// OnCounterInterrogation 总计数器请求
	OnCounterInterrogation(asdu.Connect, *asdu.ASDU, asdu.QualifierCountCall) error
	// OnRead 读定值请求
	OnRead(asdu.Connect, *asdu.ASDU, asdu.InfoObjAddr) error
	// OnClockSync 时钟同步请求
	OnClockSync(asdu.Connect, *asdu.ASDU, time.Time) error
	// OnResetProcess 进程重置请求
	OnResetProcess(asdu.Connect, *asdu.ASDU, asdu.QualifierOfResetProcessCmd) error
	// OnDelayAcquisition 延迟获取请求
	OnDelayAcquisition(asdu.Connect, *asdu.ASDU, uint16) error
	// OnASDU 控制命令请求
	OnASDU(asdu.Connect, *asdu.ASDU) error
}

func (s *ServerHandler) InterrogationHandler(conn asdu.Connect, pack *asdu.ASDU, quality asdu.QualifierOfInterrogation) error {
	return s.handler.OnInterrogation(conn, pack, quality)
}

func (s *ServerHandler) CounterInterrogationHandler(conn asdu.Connect, pack *asdu.ASDU, quality asdu.QualifierCountCall) error {
	return s.handler.OnCounterInterrogation(conn, pack, quality)
}

func (s *ServerHandler) ReadHandler(conn asdu.Connect, pack *asdu.ASDU, addr asdu.InfoObjAddr) error {
	return s.handler.OnRead(conn, pack, addr)
}

func (s *ServerHandler) ClockSyncHandler(conn asdu.Connect, pack *asdu.ASDU, time time.Time) error {
	return s.handler.OnClockSync(conn, pack, time)
}

func (s *ServerHandler) ResetProcessHandler(conn asdu.Connect, pack *asdu.ASDU, quality asdu.QualifierOfResetProcessCmd) error {
	return s.handler.OnResetProcess(conn, pack, quality)
}

func (s *ServerHandler) DelayAcquisitionHandler(conn asdu.Connect, pack *asdu.ASDU, msec uint16) error {
	return s.handler.OnDelayAcquisition(conn, pack, msec)
}

func (s *ServerHandler) ASDUHandler(conn asdu.Connect, pack *asdu.ASDU) error {
	return s.handler.OnASDU(conn, pack)
}
