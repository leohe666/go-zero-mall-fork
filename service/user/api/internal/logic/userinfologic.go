package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"mall/service/user/api/internal/svc"
	"mall/service/user/api/internal/types"
	"mall/service/user/rpc/types/user"

	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type UserInfoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUserInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UserInfoLogic {
	return &UserInfoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UserInfoLogic) UserInfo() (resp *types.UserInfoResponse, err error) {
	uid, _ := l.ctx.Value("uid").(json.Number).Int64()

	// 获取客户端 IP (handler 里已用 httpx.GetRemoteAddr 解析并存入 context)
	clientIPStr := ""
	if v := l.ctx.Value("client_ip"); v != nil {
		clientIPStr = v.(string)
	}

	// 给当前 span 补 client_ip 属性，使 Jaeger 链路直接可见客户端 IP
	if span := trace.SpanFromContext(l.ctx); span != nil {
		span.SetAttributes(attribute.String("http.client_ip", clientIPStr))
	}

	// 结构化日志：带 trace_id、client_ip，供 Loki 收集并与 Jaeger 关联
	traceID := ""
	if span := trace.SpanFromContext(l.ctx); span != nil && span.SpanContext().HasTraceID() {
		traceID = span.SpanContext().TraceID().String()
	}
	l.Infof("userinfo request received6",
		logx.Field("trace_id", traceID),
		logx.Field("client_ip", clientIPStr),
		logx.Field("uid", uid),
		logx.Field("timestamp", time.Now().Format(time.RFC3339)),
	)
	// panic("test: reached logic layer")

	// return nil, fmt.Errorf("模拟内部错误")

	fmt.Println("aaaa", http.StatusInternalServerError)
	res, err := l.svcCtx.UserRpc.UserInfo(l.ctx, &user.UserInfoRequest{
		Id: uid,
	})
	if err != nil {
		l.Errorf("userinfo rpc error",
			logx.Field("trace_id", traceID),
			logx.Field("client_ip", clientIPStr),
			logx.Field("uid", uid),
			logx.Field("error", err.Error()),
		)
		return nil, err
	}

	l.Infof("userinfo request success",
		logx.Field("trace_id", traceID),
		logx.Field("client_ip", clientIPStr),
		logx.Field("uid", uid),
	)

	return &types.UserInfoResponse{
		Id:     res.Id,
		Name:   res.Name,
		Gender: res.Gender,
		Mobile: res.Mobile,
	}, nil
}

// test hot reload
// hot reload test
// hot reload test 2
// hot reload test 3 -  2026年 8月15日 星期六 14时58分17秒 CST
// hot reload test 4 -  2026年 8月15日 星期六 15时07分33秒 CST
// hot reload test 5 -  2026年 8月15日 星期六 15时15分41秒 CST
// hot reload test 6 -  2026年 8月15日 星期六 15时21分47秒 CST
// verify hot-reload  1786778956
// test restart fix  Sat Aug 15 19:52:10 CST 2026
// test restart fix 2  Sat Aug 15 19:57:27 CST 2026
// test restart fix 3  Sat Aug 15 20:05:12 CST 2026
// test restart fix 4  Sat Aug 15 20:10:36 CST 2026
// only user_api change  Sat Aug 15 21:03:34 CST 2026
// test binary stable wait  Sat Aug 15 21:26:09 CST 2026
// test ELF wait  Sat Aug 15 21:31:24 CST 2026
// test ELF wait v2  Sat Aug 15 21:39:23 CST 2026
// final verification  Sat Aug 15 21:49:23 CST 2026
// test debounce  Sat Aug 15 21:56:10 CST 2026
