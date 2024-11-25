package iec

import (
	"github.com/thinkgos/go-iecp5/asdu"
	"log"
	"time"
	"zero-service/app/iecrpc/internal/svc"
)

type IecHandler struct {
	svcCtx *svc.ServiceContext
}

func NewIecHandler(svcCtx *svc.ServiceContext) *IecHandler {
	return &IecHandler{
		svcCtx: svcCtx,
	}
}

func (sf *IecHandler) InterrogationHandler(c asdu.Connect, asduPack *asdu.ASDU, qoi asdu.QualifierOfInterrogation) error {
	log.Println("qoi", qoi)
	// asduPack.SendReplyMirror(c, asdu.ActivationCon)
	// err := asdu.Single(c, false, asdu.CauseOfTransmission{Cause: asdu.Inrogen}, asdu.GlobalCommonAddr,
	// 	asdu.SinglePointInfo{})
	// if err != nil {
	// 	// log.Println("falied")
	// } else {
	// 	// log.Println("success")
	// }
	// asduPack.SendReplyMirror(c, asdu.ActivationTerm)
	return nil
}
func (sf *IecHandler) CounterInterrogationHandler(asdu.Connect, *asdu.ASDU, asdu.QualifierCountCall) error {
	return nil
}
func (sf *IecHandler) ReadHandler(asdu.Connect, *asdu.ASDU, asdu.InfoObjAddr) error {
	return nil
}
func (sf *IecHandler) ClockSyncHandler(asdu.Connect, *asdu.ASDU, time.Time) error {
	return nil
}
func (sf *IecHandler) ResetProcessHandler(asdu.Connect, *asdu.ASDU, asdu.QualifierOfResetProcessCmd) error {
	return nil
}
func (sf *IecHandler) DelayAcquisitionHandler(asdu.Connect, *asdu.ASDU, uint16) error {
	return nil
}
func (sf *IecHandler) ASDUHandler(asdu.Connect, *asdu.ASDU) error { return nil }
