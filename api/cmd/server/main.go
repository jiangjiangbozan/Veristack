// Package main 是 Veristack API 服务的程序入口，负责加载配置、组装路由并管理生命周期。
package main

import (
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"api/internal/acceptance"
	"api/internal/change"
	"api/internal/delivery"
	"api/internal/evidence"
	"api/internal/identity"
	"api/internal/integration"
	"api/internal/milestone"
	"api/internal/platform/ai"
	"api/internal/platform/auth"
	"api/internal/platform/config"
	"api/internal/platform/database"
	"api/internal/platform/logger"
	appMiddleware "api/internal/platform/middleware"
	"api/internal/platform/response"
	"api/internal/platform/storage"
	"api/internal/project"
	"api/internal/scope"
)

const (
	gracefulTimeout = 10 * time.Second
	healthPath      = "/healthz"
	apiPrefix       = "/api/v1"
)

// AppState 聚合服务运行所需的共享依赖，供各 Handler 注入使用。
type AppState struct {
	Auth       *auth.Manager       // 令牌管理器
	Config     *config.Config      // 全局配置
	DB         *pgxpool.Pool       // 数据库连接池
	Logger     *slog.Logger        // 结构化日志器
	Model      ai.Model            // 模型适配器
	SigningKey ed25519.PrivateKey  // 交付清单签名私钥
	Storage    storage.ObjectStore // 对象存储
}

// main 加载配置、建立数据库连接、启动服务并等待信号后优雅关闭。
func main() {
	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	log := logger.New(cfg.LogLevel)
	pool, err := prepareDatabase(ctx, cfg, log)
	if err != nil {
		panic(err)
	}
	defer pool.Close()
	signingKey, err := loadSigningKey(cfg, log)
	if err != nil {
		panic(err)
	}
	state := &AppState{
		Auth:       auth.NewManager(cfg.Auth.Secret, cfg.Auth.ExpiresIn),
		Config:     cfg,
		DB:         pool,
		Logger:     log,
		Model:      buildModel(cfg),
		SigningKey: signingKey,
		Storage:    &storage.LocalStore{Root: cfg.Storage.LocalRoot},
	}
	workerCtx, stopWorker := context.WithCancel(ctx)
	defer stopWorker()
	go (&integration.Worker{Database: pool, Logger: log}).Run(workerCtx)
	server := newHTTPServer(cfg, state)
	go listen(server, log)
	waitForShutdown(server, log)
}

// buildModel 依据配置选择模型适配器，未配置密钥时退化为本地确定性规则。
func buildModel(cfg *config.Config) ai.Model {
	if cfg.AI.APIKey == "" {
		return ai.NewHeuristicModel()
	}
	return ai.NewOpenAICompatible(cfg.AI)
}

// loadSigningKey 读取交付清单签名私钥。
//
// 采用 Fail Closed 原则：配置了私钥路径就必须加载成功，否则服务拒绝启动。
// 静默降级为不签名会让开发者误以为交付证据可核验，实际却无法验证来源。
func loadSigningKey(cfg *config.Config, log *slog.Logger) (ed25519.PrivateKey, error) {
	if cfg.Signing.PrivateKeyPath == "" {
		log.Info("未配置签名私钥，导出的 Evidence Pack 不带签名")
		return nil, nil
	}
	content, err := os.ReadFile(cfg.Signing.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("读取签名私钥失败: %w", err)
	}
	if len(content) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("签名私钥长度不合法，期望 %d 字节，实际 %d 字节", ed25519.PrivateKeySize, len(content))
	}
	return ed25519.PrivateKey(content), nil
}

// healthCheck 返回服务存活状态，供容器编排与部署检查使用。
func healthCheck(w http.ResponseWriter, _ *http.Request) {
	response.OK(w, map[string]string{"status": "ok"})
}

// listen 启动监听，非正常退出时记录错误并结束进程。
func listen(server *http.Server, log *slog.Logger) {
	log.Info("服务启动", "addr", server.Addr)
	err := server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("服务异常退出", "error", err)
		os.Exit(1)
	}
}

// newHTTPServer 组装 Chi 路由与中间件，返回配置完成的 HTTP 服务。
func newHTTPServer(cfg *config.Config, state *AppState) *http.Server {
	router := chi.NewRouter()
	router.Use(chiMiddleware.RequestID)
	router.Use(chiMiddleware.RealIP)
	router.Use(chiMiddleware.Recoverer)
	// 写请求统一开启请求级事务，业务写入与审计写入落在同一事务里。
	router.Use(database.Middleware(state.DB))
	router.Get(healthPath, healthCheck)
	router.Route(apiPrefix, func(r chi.Router) {
		publicIdentity := identity.Handler{Service: &identity.Service{Auth: state.Auth, Database: state.DB}}
		publicIdentity.Routes(r)
		acceptanceHandler := &acceptance.Handler{
			EvidenceService: &evidence.Service{Database: state.DB, Storage: state.Storage},
			Identity:        &acceptance.CustomerIdentityService{Database: state.DB, Mailer: buildMailer(cfg)},
			Service:         &acceptance.Service{Database: state.DB},
		}
		acceptanceHandler.PortalRoutes(r)
		integrationHandler := integration.Handler{Service: &integration.Service{
			AI:       state.Model,
			Database: state.DB,
			Webhook:  cfg.Webhook,
		}}
		integrationHandler.PublicRoutes(r)
		r.Group(func(protected chi.Router) {
			protected.Use(appMiddleware.Authenticate(state.Auth))
			registerRoutes(protected, state)
			publicIdentity.MeRoutes(protected)
		})
	})
	return &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}
}

// prepareDatabase 建立连接池并执行迁移，让服务启动即可使用最新结构。
func prepareDatabase(ctx context.Context, cfg *config.Config, log *slog.Logger) (*pgxpool.Pool, error) {
	pool, err := database.New(ctx, cfg.Database)
	if err != nil {
		return nil, err
	}
	if err := database.Migrate(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}
	log.Info("数据库就绪", "driver", "postgresql")
	return pool, nil
}

// registerRoutes 注册阶段一的业务路由，所有模块共享同一数据库连接池。
func registerRoutes(r chi.Router, state *AppState) {
	packRoot := filepath.Join(state.Config.Storage.LocalRoot, "packs")
	projectHandler := project.Handler{Service: &project.Service{Database: state.DB}}
	projectHandler.Routes(r)
	scopeHandler := scope.Handler{Service: &scope.Service{Database: state.DB}}
	scopeHandler.Routes(r)
	changeHandler := change.Handler{Service: &change.Service{Database: state.DB}}
	changeHandler.Routes(r)
	milestoneHandler := milestone.Handler{Service: &milestone.Service{Database: state.DB}}
	milestoneHandler.Routes(r)
	deliveryHandler := delivery.Handler{
		Service: &delivery.Service{
			Database:    state.DB,
			PackRoot:    packRoot,
			StorageRoot: state.Config.Storage.LocalRoot,
		},
		SigningKey: state.SigningKey,
	}
	deliveryHandler.Routes(r)
	evidenceHandler := evidence.Handler{Service: &evidence.Service{Database: state.DB, Storage: state.Storage}}
	evidenceHandler.Routes(r)
	acceptanceHandler := acceptance.Handler{
		EvidenceService: &evidence.Service{Database: state.DB, Storage: state.Storage},
		Service:         &acceptance.Service{Database: state.DB},
	}
	acceptanceHandler.Routes(r)
	integrationHandler := integration.Handler{Service: &integration.Service{
		AI:       state.Model,
		Database: state.DB,
		Webhook:  state.Config.Webhook,
	}}
	integrationHandler.Routes(r)
	state.Logger.Debug("业务路由注册完成")
}

// waitForShutdown 等待终止信号，并在超时内完成优雅关闭。
func waitForShutdown(server *http.Server, log *slog.Logger) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	<-signals
	ctx, cancel := context.WithTimeout(context.Background(), gracefulTimeout)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Error("优雅关闭失败", "error", err)
	}
	log.Info("服务已关闭")
}

// buildMailer 依据配置选择邮件发送方式，未配置 SMTP 时退化为日志输出。
func buildMailer(cfg *config.Config) acceptance.Mailer {
	if cfg.SMTP.Host == "" {
		return acceptance.LogMailer{}
	}
	return acceptance.NewSMTPMailer(cfg.SMTP)
}
