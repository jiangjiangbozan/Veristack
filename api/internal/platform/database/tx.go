// Package database 提供事务工作单元，使业务写入与审计写入落在同一个事务里。
package database

import (
	"bytes"
	"context"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"api/internal/platform/db"
)

type transactionKey struct{}

// WithTx 在一个事务中执行 fn。
//
// 上下文已有事务时直接参与外层事务，不会另开连接，保证嵌套调用仍然原子。
func WithTx(ctx context.Context, pool *pgxpool.Pool, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(transactionKey{}).(pgx.Tx); ok {
		return fn(ctx)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	transactionContext := context.WithValue(ctx, transactionKey{}, tx)
	if err := fn(transactionContext); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Queries 返回与当前上下文事务绑定的查询对象。
//
// 上下文中有事务时使用事务，否则回退到连接池，
// 使只读查询无需显式开启事务也能正常工作。
func Queries(ctx context.Context, pool *pgxpool.Pool) *db.Queries {
	if tx, ok := ctx.Value(transactionKey{}).(pgx.Tx); ok {
		return db.New(tx)
	}
	return db.New(pool)
}

// Middleware 为写请求开启请求级事务。
//
// 响应先写入缓冲区，事务提交成功后才真正发给客户端；
// 提交失败或处理出错都会回滚，并向客户端返回 500。
// 这样「审计写入失败」就不会留下已经生效的业务变更。
func Middleware(pool *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !isWriteMethod(r.Method) {
				next.ServeHTTP(w, r)
				return
			}
			tx, err := pool.Begin(r.Context())
			if err != nil {
				http.Error(w, "无法开启事务", http.StatusInternalServerError)
				return
			}
			defer tx.Rollback(r.Context())
			recorder := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
			transactionContext := context.WithValue(r.Context(), transactionKey{}, tx)
			next.ServeHTTP(recorder, r.WithContext(transactionContext))
			if recorder.status >= http.StatusBadRequest {
				// 业务处理失败：把错误响应发出去并回滚事务。
				recorder.flush()
				return
			}
			if err := tx.Commit(r.Context()); err != nil {
				http.Error(w, "事务提交失败，操作已回滚", http.StatusInternalServerError)
				return
			}
			recorder.flush()
		})
	}
}

// isWriteMethod 判断请求是否会修改数据，只读请求不开启事务。
func isWriteMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	default:
		return true
	}
}

// responseRecorder 缓冲响应内容，使事务提交结果先于响应送达客户端。
type responseRecorder struct {
	http.ResponseWriter
	body   bytes.Buffer
	status int
}

// Write 记录响应内容而不立即发送。
func (r *responseRecorder) Write(content []byte) (int, error) {
	return r.body.Write(content)
}

// WriteHeader 记录响应状态码。
func (r *responseRecorder) WriteHeader(status int) {
	r.status = status
}

// flush 在事务提交成功后把缓冲的响应发送给客户端。
func (r *responseRecorder) flush() {
	r.ResponseWriter.WriteHeader(r.status)
	_, _ = r.ResponseWriter.Write(r.body.Bytes())
}
