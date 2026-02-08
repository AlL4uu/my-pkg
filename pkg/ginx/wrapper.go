package ginx

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"webook/internal/web/ijwt"
	"webook/pkg/logger"
)

type HandlerFuncWithClaims[T any] func(ctx *gin.Context, req T, claims *ijwt.UserClaims) (Result, error)
type HandlerFuncNoClaims[T any] func(ctx *gin.Context, req T) (Result, error)

// WrapGeneric
// bindFunc: 绑定策略：JSON/URI/Query
// handler: 支持有 claims / 无 claims 两种 handler
func WrapGeneric[T any](l logger.LoggerV1, needAuth bool, bindFunc func(*gin.Context, *T) error, handler interface{}) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req T
		if err := bindFunc(ctx, &req); err != nil {
			l.Warn("请求参数绑定失败",
				logger.Field{Key: "path", Val: ctx.Request.URL.Path},
				logger.Field{Key: "error", Val: err.Error()},
			)
			ctx.JSON(http.StatusBadRequest, Error(400, "请求参数错误"))
			return
		}

		var claims *ijwt.UserClaims
		if needAuth {
			c, exists := ctx.Get("user")
			if !exists {
				l.Warn("用户未登录", logger.Field{Key: "path", Val: ctx.Request.URL.Path})
				ctx.JSON(http.StatusUnauthorized, Error(401, "用户未登录"))
				return
			}
			var ok bool
			claims, ok = c.(*ijwt.UserClaims)
			if !ok {
				l.Error("claims 类型断言失败", logger.Field{Key: "claims_type", Val: c})
				ctx.JSON(http.StatusInternalServerError, Error(500, "系统错误"))
				return
			}
		}

		// 执行 handler
		var res Result
		var err error

		if needAuth {
			h := handler.(HandlerFuncWithClaims[T])
			res, err = h(ctx, req, claims)
		} else {
			h := handler.(HandlerFuncNoClaims[T])
			res, err = h(ctx, req)
		}

		if err != nil {
			fields := []logger.Field{
				{Key: "path", Val: ctx.Request.URL.Path},
				{Key: "error", Val: err.Error()},
			}
			if needAuth && claims != nil {
				fields = append(fields, logger.Field{Key: "uid", Val: claims.Uid})
			}
			l.Error("业务处理失败", fields...)
			ctx.JSON(http.StatusOK, Error(500, "系统错误"))
			return
		}

		ctx.JSON(http.StatusOK, res)
	}
}

func Wrap[T any](l logger.LoggerV1, fn HandlerFuncWithClaims[T]) gin.HandlerFunc {
	return WrapGeneric(l, true, func(ctx *gin.Context, req *T) error {
		return ctx.ShouldBindJSON(req)
	}, fn)
}

func WrapNoAuth[T any](l logger.LoggerV1, fn HandlerFuncNoClaims[T]) gin.HandlerFunc {
	return WrapGeneric(l, false, func(ctx *gin.Context, req *T) error {
		return ctx.ShouldBindJSON(req)
	}, fn)
}

func WrapUri[T any](l logger.LoggerV1, fn HandlerFuncWithClaims[T]) gin.HandlerFunc {
	return WrapGeneric(l, true, func(ctx *gin.Context, req *T) error {
		return ctx.ShouldBindUri(req)
	}, fn)
}

func WrapQuery[T any](l logger.LoggerV1, fn HandlerFuncWithClaims[T]) gin.HandlerFunc {
	return WrapGeneric(l, true, func(ctx *gin.Context, req *T) error {
		return ctx.ShouldBindQuery(req)
	}, fn)
}

func WrapQueryNoAuth[T any](l logger.LoggerV1, fn HandlerFuncNoClaims[T]) gin.HandlerFunc {
	return WrapGeneric(l, false, func(ctx *gin.Context, req *T) error {
		return ctx.ShouldBindQuery(req)
	}, fn)
}
