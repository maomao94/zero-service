# zero-service

####
go build -o ./app/zerorpc ./zerorpc/zerorpc.go
go build -o ./app/zeroalarm ./zeroalarm/zeroalarm.go

#### 相关包
https://github.com/hibiken/asynq/
https://github.com/Masterminds/squirrel

#### util
docker工具 本地快速执行 docker 相关命令
go build -o ./util/dockeru/dk ./util/dockeru/main.go