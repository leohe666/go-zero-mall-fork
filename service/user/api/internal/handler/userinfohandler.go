package handler

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"mall/service/user/api/internal/logic"
	"mall/service/user/api/internal/svc"
)

func UserInfoHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 提取客户端 IP 放入 context，供 logic 记录日志
		ctx := r.Context()
		clientIP := httpx.GetRemoteAddr(r)
		ctx = context.WithValue(ctx, "client_ip", clientIP)

		l := logic.NewUserInfoLogic(ctx, svcCtx)
		resp, err := l.UserInfo()
		if err != nil {
			httpx.Error(w, err)
		} else {
			httpx.OkJson(w, resp)
		}
	}
}
