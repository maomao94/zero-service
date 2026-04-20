---
trigger: always_on
alwaysApply: true
---
git 提交示例需要用中文
这个一个基于 go-zero 的项目
每个业务服务都有一个 gen.sh 来执行基础代码生成
网关的接口必须先更改 .api 文件，再执行 gen.sh
grpc 服务必须先更改 proto 文件，再执行 gen.sh
api 描述 请求和响应都是 request 和 response
grpc 描述 请求和响应都是 req  和 res 
每个请求和响应都是成对出现， 例如 chatReq 就会有 chatRes
ai 相关的业务 基于字节跳动 eino 框架开发