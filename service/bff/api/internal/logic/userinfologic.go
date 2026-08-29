// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"
	"encoding/json"
	"fmt"

	"mall/service/bff/api/internal/svc"
	"mall/service/bff/api/internal/types"
	"mall/service/user/rpc/types/user"

	"github.com/zeromicro/go-zero/core/logx"
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
	// 从 JWT 中间件注入的 context 取 uid（go-zero 存的是 json.Number）。
	// 用 comma-ok 断言 + 处理错误，避免类型不符时 panic、或 uid 静默变 0。
	uidNumber, ok := l.ctx.Value("uid").(json.Number)
	if !ok {
		return nil, fmt.Errorf("uid not found or invalid in context")
	}
	uid, err := uidNumber.Int64()
	if err != nil {
		return nil, fmt.Errorf("invalid uid in context: %w", err)
	}

	res, err := l.svcCtx.UserRpc.UserInfo(l.ctx, &user.UserInfoRequest{
		Id: uid,
	})
	if err != nil {
		return nil, err
	}

	return &types.UserInfoResponse{
		Id:     res.Id,
		Name:   res.Name,
		Gender: res.Gender,
		Mobile: res.Mobile,
	}, nil
}
