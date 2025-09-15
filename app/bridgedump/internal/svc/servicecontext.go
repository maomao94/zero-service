package svc

import (
	"context"
	"fmt"
	"path/filepath"
	"zero-service/app/bridgedump/internal/config"
	"zero-service/model"

	"github.com/dromara/carbon/v2"
	"github.com/duke-git/lancet/v2/fileutil"
	"github.com/zeromicro/go-zero/core/jsonx"
	"github.com/zeromicro/go-zero/core/trace"
)

type ServiceContext struct {
	Config config.Config
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config: c,
	}
}

func (*ServiceContext) DumpBridgeData(ctx context.Context, dumpPath, subDir string, in any) (string, error) {
	traceID := trace.TraceIDFromContext(ctx)
	// 确保目录存在
	mkdirPath := filepath.Join(dumpPath, subDir)
	if err := fileutil.CreateDir(mkdirPath); err != nil {
		return "", err
	}
	// 请求体序列化
	bridgeBody, err := jsonx.Marshal(in)
	if err != nil {
		return "", err
	}
	now := carbon.Now()
	filename := fmt.Sprintf("%s_%s_json.txt", now.Format("Ymd_His"), traceID)
	writeFilePath := filepath.Join(dumpPath, subDir, filename)
	// 包装消息体
	message := model.BridgeMsgBody{
		TraceId:  traceID,
		Body:     string(bridgeBody),
		Time:     now.ToDateTimeMicroString(),
		FilePath: writeFilePath,
	}
	logJson, err := jsonx.Marshal(message)
	if err != nil {
		return "", err
	}
	// 计算 JSON 字节长度
	size := len(logJson)
	// 构造完整文件内容
	fileContent := fmt.Sprintf(`<!System=OMG Version=1.05 Code=utf-8 Data=1.0!>
<Bridge:=Free Size=%d>
%s
</Bridge:=Free>
`, size, string(logJson))
	if err := fileutil.WriteStringToFile(writeFilePath, fileContent, true); err != nil {
		return "", err
	}
	return writeFilePath, nil
}
