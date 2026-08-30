// Package response 提供统一 HTTP 响应格式的全局包装中间件。
//
// 所有接口（透传 RPC + 自定义聚合 handler）的响应统一为：
//
//	成功: HTTP 2xx + {"code":0,"msg":"ok","data":<业务数据>}
//	失败: HTTP 4xx/5xx + {"code":<业务码或HTTP码>,"msg":"<错误信息>","data":null}
//
// 说明：
//   - 401（JWT 未登录/token 失效）保持 go-zero 内置中间件的默认行为（HTTP 401 + 空 body），
//     不参与统一包装；
//   - 响应头附带 request_id（透传客户端 X-Request-Id，否则取 Traceparent 的 trace-id，
//     再否则随机生成），与 Loki/Jaeger 链路追踪的 trace-id 对应，可按它查全链路日志。
//
// 使用方式（统一网关入口）：
//
//	gw.Server.Use(response.Wrapper)
package response

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"go.opentelemetry.io/otel/trace"
)

// Body 统一响应结构
type Body struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// rpcErrorPattern 匹配 go-zero gRPC 透传错误文本：
// "rpc error: code = Code(100) desc = 该用户已存在"
var rpcErrorPattern = regexp.MustCompile(`rpc error: code = Code\((\d+)\) desc = (.*)`)

// Wrapper rest.Middleware：包装所有响应的 body 为统一 {code,msg,data} 结构，
// 并附带 request_id 响应头（供按 trace 查日志）。
func Wrapper(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 响应头附带 request_id：客户端 X-Request-Id -> Traceparent trace-id -> 随机
		rid := resolveRequestID(r)
		if rid != "" {
			w.Header().Set("X-Request-Id", rid)
			w.Header().Set("request_id", rid)
		}

		rw := &respWriter{ResponseWriter: w}
		next(rw, r)
		rw.wrap()
	}
}

// respWriter 缓冲底层响应，延迟到 handler 执行完后统一改写
type respWriter struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

func (w *respWriter) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
	}
}

func (w *respWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.body.Write(p)
}

// wrap 把缓冲的响应改写为统一格式后写出
func (w *respWriter) wrap() {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	status := w.status
	raw := w.body.Bytes()

	// 已是统一格式（code/msg/data 三键齐全）则不重复包装
	var m map[string]interface{}
	if json.Unmarshal(raw, &m) == nil {
		_, hasCode := m["code"]
		_, hasMsg := m["msg"]
		_, hasData := m["data"]
		if hasCode && hasMsg && hasData {
			w.write(status, raw)
			return
		}
	}

	var resp Body
	if status < http.StatusBadRequest {
		resp = Body{Code: 0, Msg: "ok", Data: parseData(raw)}
	} else {
		code, msg := parseError(status, raw)
		resp = Body{Code: code, Msg: msg, Data: nil}
	}

	out, err := json.Marshal(resp)
	if err != nil {
		out = []byte(`{"code":-1,"msg":"internal marshal error","data":null}`)
	}
	w.write(status, out)
}

func (w *respWriter) write(status int, body []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.ResponseWriter.WriteHeader(status)
	_, _ = w.ResponseWriter.Write(body)
}

// parseData 解析成功响应体：JSON 原样保留，非 JSON 转字符串
func parseData(raw []byte) interface{} {
	var v interface{}
	if err := json.Unmarshal(raw, &v); err != nil {
		// 空 body 视为 null
		if len(bytes.TrimSpace(raw)) == 0 {
			return nil
		}
		return string(raw)
	}
	return v
}

// parseError 从错误响应中提取业务码与错误信息
func parseError(status int, raw []byte) (int, string) {
	// 1) gRPC 透传错误文本: rpc error: code = Code(100) desc = xxx
	if m := rpcErrorPattern.FindStringSubmatch(string(raw)); len(m) == 3 {
		code, _ := strconv.Atoi(m[1])
		return code, m[2]
	}
	// 2) JSON 错误: {"code":...,"message":...} 或 {"code":...,"desc":...}
	var m map[string]interface{}
	if json.Unmarshal(raw, &m) == nil {
		msg, _ := m["message"].(string)
		if msg == "" {
			msg, _ = m["desc"].(string)
		}
		if code, ok := m["code"].(float64); ok && code != 0 {
			return int(code), msg
		}
		if msg != "" {
			return status, msg
		}
	}
	// 3) 兜底
	return status, string(raw)
}

// resolveRequestID 解析请求对应的 request_id（用于响应头 + 全链路日志关联）：
//  1. 客户端显式传入的 X-Request-Id（透传）
//  2. go-zero 链路追踪上下文里的 trace-id（与响应头 Traceparent 一致，Loki 日志 trace 字段同源）
//  3. W3C Traceparent 请求头里的 trace-id
//  4. 随机生成 16 位 hex
func resolveRequestID(r *http.Request) string {
	if id := r.Header.Get("X-Request-Id"); id != "" {
		return id
	}
	if sc := trace.SpanContextFromContext(r.Context()); sc.HasTraceID() {
		return sc.TraceID().String()
	}
	if tp := r.Header.Get("Traceparent"); tp != "" {
		parts := strings.Split(tp, "-")
		if len(parts) >= 2 && parts[0] == "00" && len(parts[1]) == 32 {
			return parts[1]
		}
	}
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}
