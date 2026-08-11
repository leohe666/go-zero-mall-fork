package logic

import (
	"context"
	"encoding/json"
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
	l.Infof("userinfo request received",
		logx.Field("trace_id", traceID),
		logx.Field("client_ip", clientIPStr),
		logx.Field("uid", uid),
		logx.Field("timestamp", time.Now().Format(time.RFC3339)),
	)

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