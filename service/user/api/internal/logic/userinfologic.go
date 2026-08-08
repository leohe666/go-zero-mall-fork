package logic

import (
	"context"
	"encoding/json"
	"time"

	"mall/service/user/api/internal/svc"
	"mall/service/user/api/internal/types"
	"mall/service/user/rpc/types/user"

	"github.com/zeromicro/go-zero/core/logx"
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

	// 结构化日志：带 trace_id，供 Loki 收集并与 Jaeger 关联
	traceID := ""
	if span := trace.SpanFromContext(l.ctx); span != nil && span.SpanContext().HasTraceID() {
		traceID = span.SpanContext().TraceID().String()
	}
	l.Infof("userinfo request received",
		logx.Field("trace_id", traceID),
		logx.Field("uid", uid),
		logx.Field("timestamp", time.Now().Format(time.RFC3339)),
	)

	res, err := l.svcCtx.UserRpc.UserInfo(l.ctx, &user.UserInfoRequest{
		Id: uid,
	})
	if err != nil {
		l.Errorf("userinfo rpc error",
			logx.Field("trace_id", traceID),
			logx.Field("uid", uid),
			logx.Field("error", err.Error()),
		)
		return nil, err
	}

	l.Infof("userinfo request success",
		logx.Field("trace_id", traceID),
		logx.Field("uid", uid),
	)

	return &types.UserInfoResponse{
		Id:     res.Id,
		Name:   res.Name,
		Gender: res.Gender,
		Mobile: res.Mobile,
	}, nil
}