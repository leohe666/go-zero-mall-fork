// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package handler

import (
	"net/http"

	"mall/service/bff/api/internal/logic"
	"mall/service/bff/api/internal/svc"
	"mall/service/bff/api/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func OrderAggregateHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.OrderAggregateRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := logic.NewOrderAggregateLogic(r.Context(), svcCtx)
		resp, err := l.OrderAggregate(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
