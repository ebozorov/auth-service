# Lokal statik sozlamalar config/.env.local dan olinadi va faqat undagi kalitlar eksport qilinadi.
# Dinamik sozlamalar config/config.yaml da, ularni server o'zi o'qiydi va kuzatadi.
ENV_FILE := config/.env.local
-include $(ENV_FILE)
ENV_VARS := $(shell sed -n 's/^\([A-Za-z_][A-Za-z0-9_]*\)=.*/\1/p' $(ENV_FILE) 2>/dev/null)
ifneq ($(ENV_VARS),)
export $(ENV_VARS)
endif

BIN_DIR  := bin
BIN      := $(BIN_DIR)/auth-service
PID_FILE := $(BIN_DIR)/auth-service.pid
LOG_DIR  := logs
LOG_FILE := $(LOG_DIR)/auth-service.log

PRIVATE_KEY := $(or $(JWT_PRIVATE_KEY_PATH),certs/private.pem)
KEYS_DIR    := $(dir $(PRIVATE_KEY))

COMPOSE_FILE := docker/compose.yaml

# Vite dev server manzili (web/vite.config.ts dagi port bilan bir xil)
WEB_DEV_URL := http://localhost:5173

SWAGGER_UI_VERSION   := 5.33.0
SWAGGER_UI_INTEGRITY := sha512-wpdK+m6BU5yj6pmUdMskZVTSWYG4DLglAx3sIhylloY37i8O37IrH+YEpqdXNfpaTGxILRBFzUqLF2jKqbfI7A==
SWAGGER_UI_DIR       := swagger-ui
SWAGGER_UI_FILES     := index.html index.css swagger-ui.css swagger-ui-bundle.js swagger-ui-standalone-preset.js favicon-16x16.png favicon-32x32.png

RUNNING := [ -f $(PID_FILE) ] && kill -0 $$(cat $(PID_FILE)) 2>/dev/null

.PHONY: build dev start stop reload status services-up services-down services-status keys swagger-ui web-install web-dev web-build web-types test

# web/dist binary ichiga embed qilinadi, shuning uchun avval web build bo'ladi
build: web-build
	go build -o $(BIN) ./cmd/server

keys: $(PRIVATE_KEY)

# Live rejim: Go server air orqali (.air.toml, Go kodi o'zgarsa qayta ishga tushadi) va
# Vite dev server (HMR). Web $(WEB_DEV_URL) dan ochiladi, emaildagi link ham shu yerga keladi
# (Vite /api ni Go serverga uzatadi). Ctrl+C ikkalasini to'xtatadi.
dev: keys web/dist/index.html
	@if $(RUNNING); then echo "auth-service fonda ishlayapti, avval: make stop"; exit 1; fi
	@PUBLIC_URL=$(WEB_DEV_URL) go tool -modfile=tools/go.mod air & pid=$$!; \
	trap 'kill $$pid 2>/dev/null; wait $$pid' EXIT INT TERM; \
	cd web && npm run dev

# web/dist binary ichiga embed qilinadi, u bo'lmasa go build ishlamaydi
web/dist/index.html:
	$(MAKE) --no-print-directory web-build

# Web (web/): React SPA, build natijasi web/dist ga tushadi va binary ichiga embed qilinadi
web-install:
	cd web && npm ci

# Vite dev server (:5173), /api va /.well-known so'rovlari Go serverga proxy qilinadi
web-dev:
	cd web && npm run dev

web-build:
	cd web && npm run build

# docs/openapi.json dan web/src/api/schema.d.ts ni qayta yaratadi
web-types:
	cd web && npm run gen:api

$(PRIVATE_KEY):
	@mkdir -p $(KEYS_DIR)
	openssl genpkey -quiet -algorithm RSA -pkeyopt rsa_keygen_bits:2048 -out $@
	chmod 600 $@

# swagger-ui-dist ni npm dan yuklaydi. swagger-initializer.js bizniki, u ustidan yozilmaydi.
swagger-ui:
	@tmp=$$(mktemp -d) && trap 'rm -rf $$tmp' EXIT && \
	curl -fsSL -o $$tmp/pkg.tgz https://registry.npmjs.org/swagger-ui-dist/-/swagger-ui-dist-$(SWAGGER_UI_VERSION).tgz && \
	actual="sha512-$$(openssl dgst -sha512 -binary $$tmp/pkg.tgz | base64 -w0)" && \
	if [ "$$actual" != "$(SWAGGER_UI_INTEGRITY)" ]; then echo "Integrity xato: $$actual"; exit 1; fi && \
	tar -xzf $$tmp/pkg.tgz -C $$tmp && \
	mkdir -p $(SWAGGER_UI_DIR) && \
	for f in $(SWAGGER_UI_FILES); do cp $$tmp/package/$$f $(SWAGGER_UI_DIR)/; done && \
	echo "Swagger UI $(SWAGGER_UI_VERSION) yuklandi: $(SWAGGER_UI_DIR)/"

start: build keys
	@mkdir -p $(LOG_DIR)
	@if $(RUNNING); then \
		echo "auth-service allaqachon ishlayapti (PID $$(cat $(PID_FILE)))"; \
	else \
		nohup ./$(BIN) > $(LOG_FILE) 2>&1 & echo $$! > $(PID_FILE); \
		sleep 1; \
		if $(RUNNING); then \
			echo "auth-service ishga tushdi (PID $$(cat $(PID_FILE))), log: $(LOG_FILE)"; \
			grep -E "ishlayapti: |web: |api: |swagger ui: " $(LOG_FILE) | sed -E 's/^[0-9/]+ [0-9:]+ /  /'; \
		else \
			echo "auth-service ishga tushmadi:"; tail -n 20 $(LOG_FILE); rm -f $(PID_FILE); exit 1; \
		fi; \
	fi

stop:
	@if $(RUNNING); then \
		pid=$$(cat $(PID_FILE)); kill $$pid; \
		while kill -0 $$pid 2>/dev/null; do sleep 0.1; done; \
		rm -f $(PID_FILE); echo "auth-service to'xtatildi"; \
	else \
		rm -f $(PID_FILE); echo "auth-service ishlamayapti"; \
	fi

# Graceful restart: joriy so'rovlar tugaydi, server yangi env bilan ko'tariladi.
# config/config.yaml o'zgarishi uchun restart kerak emas.
reload:
	@$(MAKE) --no-print-directory stop start

status:
	@if $(RUNNING); then \
		echo "auth-service ishlayapti (PID $$(cat $(PID_FILE)))"; \
	else \
		echo "auth-service ishlamayapti"; \
	fi

# Tashqi servicelar: ro'yxati $(COMPOSE_FILE) da
services-up:
	docker compose -f $(COMPOSE_FILE) up -d --wait

services-down:
	docker compose -f $(COMPOSE_FILE) down

services-status:
	docker compose -f $(COMPOSE_FILE) ps

# Testlar config/.env.local ga bog'liq emas: undagi o'zgaruvchilar test process'iga berilmaydi.
# INTEGRATION=1 bo'lsa integration testlar ham ishlaydi (testcontainers, Docker kerak).
test:
	@env $(addprefix -u ,$(ENV_VARS)) go test $(if $(INTEGRATION),-tags=integration) ./...
