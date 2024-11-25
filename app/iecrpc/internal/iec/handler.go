package iec

import (
	"github.com/wendy512/go-iecp5/asdu"
	"time"
)

type ServerHandler struct {
	h CommandHandler
}

func (s *ServerHandler) InterrogationHandler(conn asdu.Connect, pack *asdu.ASDU, quality asdu.QualifierOfInterrogation) error {
	return s.h.OnInterrogation(conn, pack, quality)
}

func (s *ServerHandler) CounterInterrogationHandler(conn asdu.Connect, pack *asdu.ASDU, quality asdu.QualifierCountCall) error {
	return s.h.OnCounterInterrogation(conn, pack, quality)
}

func (s *ServerHandler) ReadHandler(conn asdu.Connect, pack *asdu.ASDU, addr asdu.InfoObjAddr) error {
	return s.h.OnRead(conn, pack, addr)
}

func (s *ServerHandler) ClockSyncHandler(conn asdu.Connect, pack *asdu.ASDU, time time.Time) error {
	return s.h.OnClockSync(conn, pack, time)
}

func (s *ServerHandler) ResetProcessHandler(conn asdu.Connect, pack *asdu.ASDU, quality asdu.QualifierOfResetProcessCmd) error {
	return s.h.OnResetProcess(conn, pack, quality)
}

func (s *ServerHandler) DelayAcquisitionHandler(conn asdu.Connect, pack *asdu.ASDU, msec uint16) error {
	return s.h.OnDelayAcquisition(conn, pack, msec)
}

func (s *ServerHandler) ASDUHandler(conn asdu.Connect, pack *asdu.ASDU) error {
	return s.h.OnASDU(conn, pack)
}
