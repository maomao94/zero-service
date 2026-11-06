package logic

import (
	"net"
	"strconv"
	"strings"
	"unicode/utf8"

	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/common/errx"
	"zero-service/common/modbusx"
	"zero-service/third_party/extproto"
)

const maxConfigBatchSize = 500

// validateConfigText 拒绝空白值和超长值，保持编码、ID 原值用于精确匹配。
func validateConfigText(field, value string, maxLength int) error {
	if strings.TrimSpace(value) == "" {
		return errx.NewErrorByPbCode(extproto.Code__1_00_PARAM_INVALID, "%s 不能为空", field)
	}
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) > maxLength {
		return errx.NewErrorByPbCode(extproto.Code__1_00_PARAM_INVALID, "%s 长度不能超过 %d 个字符且必须为有效 UTF-8", field, maxLength)
	}
	return nil
}

func validateConfigBatch(field string, values []string, maxLength int) error {
	if len(values) == 0 || len(values) > maxConfigBatchSize {
		return errx.NewErrorByPbCode(extproto.Code__1_00_PARAM_INVALID, "%s 数量必须在 [1, %d] 范围内", field, maxConfigBatchSize)
	}
	for i, value := range values {
		if err := validateConfigText(field+"["+strconv.Itoa(i)+"]", value, maxLength); err != nil {
			return err
		}
	}
	return nil
}

func validateSaveConfig(in *bridgemodbus.SaveConfigReq) error {
	if err := validateConfigText("modbusCode", in.GetModbusCode(), 128); err != nil {
		return err
	}
	if err := validateConfigText("slaveAddress", in.GetSlaveAddress(), 64); err != nil {
		return err
	}
	host, port, err := net.SplitHostPort(in.SlaveAddress)
	portNumber, portErr := strconv.Atoi(port)
	if err != nil || host == "" || strings.ContainsAny(host, " \t\r\n/\\") ||
		portErr != nil || portNumber < 1 || portNumber > 65535 || strings.IndexFunc(port, func(r rune) bool { return r < '0' || r > '9' }) >= 0 {
		return errx.NewErrorByPbCode(extproto.Code__1_00_PARAM_INVALID, "slaveAddress 必须为有效的 host:port 地址，端口范围 [1, 65535]")
	}
	if in.Slave < modbusx.MinSlave || in.Slave > modbusx.MaxSlave {
		return errx.NewErrorByPbCode(extproto.Code__1_00_PARAM_INVALID, "slave 超出范围 [%d, %d]（当前 %d）", modbusx.MinSlave, modbusx.MaxSlave, in.Slave)
	}
	return nil
}
