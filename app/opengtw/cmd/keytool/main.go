// keytool 是 opengtw 的本地联调工具：
//
//  1. reveal：直接读凭据库，用配置里的主密钥解密并输出当前生效的 signKey；
//  2. sign：用同一把 signKey 生成签名 body、X-Sign 和可直接执行的 curl；
//  3. sign -send：把签名请求真正发到网关，形成“查看 → 签名 → 调用”的本地闭环。
//
// 它不经过 HTTP 管理面，也不依赖 Redis/gRPC，适合在服务器或开发机直接排障。
// signKey 是敏感凭据：key 只写 stdout，提示和错误写 stderr。
package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"zero-service/app/opengtw/internal/config"
	"zero-service/app/opengtw/internal/security"
	"zero-service/app/opengtw/internal/signkey"
	"zero-service/common/gormx"
	"zero-service/model/gormmodel"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/conf"
)

const (
	defaultConfigFile = "app/opengtw/etc/opengtw.yaml"
	defaultGatewayURL = "http://127.0.0.1:11005"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "reveal":
		runReveal(os.Args[2:])
	case "sign":
		runSign(os.Args[2:])
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "未知子命令 %q\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `keytool - 本地查看签名密钥并生成签名请求

用法:
  keytool reveal -f app/opengtw/etc/opengtw.yaml -app app_xxx
  keytool sign   -f app/opengtw/etc/opengtw.yaml -app app_xxx \
                 -method pod.list -data '{"limit":10,"offset":0}' \
                 [-secret sec_xxx] [-send]

子命令:
  reveal  从数据库解密并输出当前生效的 signKey（仅 stdout 输出 key）
  sign    生成签名 body / bodyHash / canonical / X-Sign / curl；
          加 -send 可直接调用网关（需要 -secret）

注意:
  signKey 是敏感凭据，请勿写入日志、工单或提交到仓库。
`)
}

func runReveal(args []string) {
	fs := flag.NewFlagSet("reveal", flag.ExitOnError)
	configFile := fs.String("f", defaultConfigFile, "opengtw 配置文件")
	appID := fs.String("app", "", "应用 ID，例如 app_xxx")
	_ = fs.Parse(args)

	if strings.TrimSpace(*appID) == "" {
		fatal("缺少 -app")
	}

	c := mustLoadConfig(*configFile)
	key, err := revealKey(c, *appID)
	if err != nil {
		fatal("读取 signKey 失败: %v", err)
	}

	fmt.Fprintln(os.Stderr, "warning: signKey 是敏感凭据，请勿写入日志或提交到仓库")
	fmt.Println(key)
}

func runSign(args []string) {
	fs := flag.NewFlagSet("sign", flag.ExitOnError)
	configFile := fs.String("f", defaultConfigFile, "opengtw 配置文件")
	appID := fs.String("app", "", "应用 ID，例如 app_xxx")
	method := fs.String("method", "", "开放 method，例如 pod.list")
	data := fs.String("data", "", "data 的 JSON，例如 '{\"limit\":10}'；省略表示无参数")
	tid := fs.String("tid", "", "链路 ID；省略则不放入 body")
	timestamp := fs.Int64("timestamp", 0, "Unix 毫秒时间戳；省略取当前时间")
	nonce := fs.String("nonce", "", "防重放 nonce；省略自动生成 UUID")
	secret := fs.String("secret", "", "Basic 密码 sec_xxx；仅用于打印 curl 或 -send")
	baseURL := fs.String("url", defaultGatewayURL, "网关地址，例如 http://127.0.0.1:11005")
	send := fs.Bool("send", false, "签名后直接发送请求（需要 -secret）")
	timeout := fs.Duration("timeout", 30*time.Second, "发送请求超时时间")
	_ = fs.Parse(args)

	if strings.TrimSpace(*appID) == "" {
		fatal("缺少 -app")
	}
	if strings.TrimSpace(*method) == "" {
		fatal("缺少 -method")
	}
	rawData := strings.TrimSpace(*data)
	if rawData != "" && !json.Valid([]byte(rawData)) {
		fatal("-data 不是合法 JSON: %s", rawData)
	}

	c := mustLoadConfig(*configFile)
	key, err := revealKey(c, *appID)
	if err != nil {
		fatal("读取 signKey 失败: %v", err)
	}

	ts := *timestamp
	if ts == 0 {
		ts = time.Now().UnixMilli()
	}
	n := strings.TrimSpace(*nonce)
	if n == "" {
		n = uuid.NewString()
	}

	body, err := buildBody(*method, rawData, *tid, ts, n)
	if err != nil {
		fatal("生成请求 body 失败: %v", err)
	}

	input := &security.SignatureInput{
		AppID:     *appID,
		Timestamp: ts,
		Nonce:     n,
		Path:      security.SignatureGatewayPath,
		BodyHash:  security.HashBody(body),
	}
	sign := input.ComputeSign(key)
	endpoint := normalizeBaseURL(*baseURL) + security.SignatureGatewayPath

	fmt.Printf("appId     : %s\n", *appID)
	fmt.Printf("endpoint  : %s\n", endpoint)
	fmt.Printf("body      : %s\n", body)
	fmt.Printf("bodyHash  : %s\n", input.BodyHash)
	fmt.Printf("canonical :\n%s\n", input.CanonicalString())
	fmt.Printf("X-Sign    : v1:%s\n", sign)
	fmt.Printf("curl      : %s\n", buildCurl(endpoint, *appID, *secret, sign, string(body)))

	if !*send {
		return
	}
	if *secret == "" {
		fatal("-send 需要同时提供 -secret，用于 Basic 鉴权")
	}
	if _, err := url.ParseRequestURI(endpoint); err != nil {
		fatal("网关地址不合法: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(string(body)))
	if err != nil {
		fatal("创建请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Sign", "v1:"+sign)
	req.SetBasicAuth(*appID, *secret)

	resp, err := (&http.Client{Timeout: *timeout}).Do(req)
	if err != nil {
		fatal("调用网关失败: %v", err)
	}
	defer resp.Body.Close()

	fmt.Printf("\nresponse  : HTTP %d\n", resp.StatusCode)
	if requestID := resp.Header.Get("X-Request-Id"); requestID != "" {
		fmt.Printf("requestId : %s\n", requestID)
	}
	if _, err := io.Copy(os.Stdout, resp.Body); err != nil {
		fatal("读取响应失败: %v", err)
	}
	fmt.Println()
}

// revealKey 打开凭据库并读取应用当前生效的 signKey。
func revealKey(c config.Config, appID string) (string, error) {
	db, err := gormx.OpenWithConf(c.DB)
	if err != nil {
		return "", err
	}
	defer closeDB(db)

	return signkey.Reveal(
		context.Background(),
		gormmodel.NewOpenAppStore(db),
		gormmodel.NewOpenAppSecretStore(db),
		c.SignKeyEncryptionKey,
		appID,
	)
}

// buildBody 按固定字段顺序拼出实际发送的 body 字节。
//
// 不用 json.Marshal 包一层 struct，是为了让 data 保持用户传入的原始 JSON，
// 避免 RawMessage 被重新编码后与预期字节不一致；签名覆盖的就是这里返回的字节。
func buildBody(method, data, tid string, timestamp int64, nonce string) ([]byte, error) {
	methodJSON, err := json.Marshal(method)
	if err != nil {
		return nil, err
	}
	nonceJSON, err := json.Marshal(nonce)
	if err != nil {
		return nil, err
	}

	var b strings.Builder
	b.WriteString(`{"method":`)
	b.Write(methodJSON)
	if data != "" {
		b.WriteString(`,"data":`)
		b.WriteString(data)
	}
	if tid != "" {
		tidJSON, err := json.Marshal(tid)
		if err != nil {
			return nil, err
		}
		b.WriteString(`,"tid":`)
		b.Write(tidJSON)
	}
	fmt.Fprintf(&b, `,"timestamp":%d,"nonce":%s}`, timestamp, nonceJSON)
	return []byte(b.String()), nil
}

func buildCurl(endpoint, appID, secret, sign, body string) string {
	var b strings.Builder
	b.WriteString("curl -sS -X POST ")
	b.WriteString(shellQuote(endpoint))
	b.WriteString(" -H 'Content-Type: application/json'")
	if secret != "" {
		basic := base64.StdEncoding.EncodeToString([]byte(appID + ":" + secret))
		b.WriteString(" -H ")
		b.WriteString(shellQuote("Authorization: Basic " + basic))
	} else {
		b.WriteString(" -H 'Authorization: Basic <base64(appId:secret)>'")
	}
	b.WriteString(" -H ")
	b.WriteString(shellQuote("X-Sign: v1:" + sign))
	b.WriteString(" --data-binary ")
	b.WriteString(shellQuote(body))
	return b.String()
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// normalizeBaseURL 允许用户只写 host:port，默认补 http://；同时去掉末尾斜杠。
func normalizeBaseURL(value string) string {
	value = strings.TrimRight(strings.TrimSpace(value), "/")
	if value == "" {
		return defaultGatewayURL
	}
	if !strings.Contains(value, "://") {
		return "http://" + value
	}
	return value
}

func mustLoadConfig(path string) config.Config {
	var c config.Config
	if err := conf.Load(path, &c); err != nil {
		fatal("加载配置 %s 失败: %v", path, err)
	}
	return c
}

func closeDB(db *gormx.DB) {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return
	}
	_ = sqlDB.Close()
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}
