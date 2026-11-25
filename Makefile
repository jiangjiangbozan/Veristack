.PHONY: help api-build api-run api-vet api-test cli-build web-install web-build web-lint web-dev sqlc-gen compose-up compose-down

help:
	@echo "可用命令:"
	@echo "  api-build    编译 API 服务"
	@echo "  api-run      本地运行 API 服务"
	@echo "  api-vet      运行 go vet 静态检查"
	@echo "  cli-build    编译 Evidence Pack 核验 CLI"
	@echo "  web-install  安装前端依赖"
	@echo "  web-build    构建前端产物"
	@echo "  web-lint     运行前端 ESLint"
	@echo "  web-dev      启动前端开发服务器"
	@echo "  sqlc-gen     根据 SQL 生成类型安全代码"
	@echo "  compose-up   启动完整容器环境"
	@echo "  compose-down 停止容器环境"

api-build:
	cd api && go build ./...

api-run:
	cd api && go run ./cmd/server

api-vet:
	cd api && go vet ./...

api-test:
	cd api && go test ./...

cli-build:
	cd api && go build -o veristack ./cmd/veristack

web-install:
	cd web && npm install

web-build:
	cd web && npm run build

web-lint:
	cd web && npm run lint

web-dev:
	cd web && npm run dev

sqlc-gen:
	cd api && sqlc generate

compose-up:
	docker compose -f deploy/compose/docker-compose.yml up -d --build

compose-down:
	docker compose -f deploy/compose/docker-compose.yml down
