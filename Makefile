# Lokal sozlamalar .env.local dan olinadi va faqat undagi kalitlar eksport qilinadi
-include .env.local
ENV_VARS := $(shell sed -n 's/^\([A-Za-z_][A-Za-z0-9_]*\)=.*/\1/p' .env.local 2>/dev/null)
ifneq ($(ENV_VARS),)
export $(ENV_VARS)
endif

BIN_DIR  := bin
BIN      := $(BIN_DIR)/auth-api
PID_FILE := $(BIN_DIR)/auth-api.pid
LOG_DIR  := logs
LOG_FILE := $(LOG_DIR)/auth-api.log

PRIVATE_KEY := $(or $(JWT_PRIVATE_KEY_PATH),certs/private.pem)
KEYS_DIR    := $(dir $(PRIVATE_KEY))

COMPOSE_FILE := docker/compose.yaml

SWAGGER_UI_VERSION   := 5.33.0
SWAGGER_UI_INTEGRITY := sha512-wpdK+m6BU5yj6pmUdMskZVTSWYG4DLglAx3sIhylloY37i8O37IrH+YEpqdXNfpaTGxILRBFzUqLF2jKqbfI7A==
SWAGGER_UI_DIR       := swagger-ui
SWAGGER_UI_FILES     := index.html index.css swagger-ui.css swagger-ui-bundle.js swagger-ui-standalone-preset.js favicon-16x16.png favicon-32x32.png

RUNNING := [ -f $(PID_FILE) ] && kill -0 $$(cat $(PID_FILE)) 2>/dev/null

.PHONY: build start stop reload status services-up services-down services-status keys swagger-ui web-install web-dev web-build web-types test

# web/dist binary ichiga embed qilinadi, shuning uchun avval web build bo'ladi
build: web-build
	go build -o $(BIN) ./cmd/server

keys: $(PRIVATE_KEY)

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
		echo "auth-api allaqachon ishlayapti (PID $$(cat $(PID_FILE)))"; \
	else \
		nohup ./$(BIN) > $(LOG_FILE) 2>&1 & echo $$! > $(PID_FILE); \
		sleep 1; \
		if $(RUNNING); then \
			echo "auth-api ishga tushdi (PID $$(cat $(PID_FILE))), log: $(LOG_FILE)"; \
			grep -E "ishlayapti: |web: |api: |swagger ui: " $(LOG_FILE) | sed -E 's/^[0-9/]+ [0-9:]+ /  /'; \
		else \
			echo "auth-api ishga tushmadi:"; tail -n 20 $(LOG_FILE); rm -f $(PID_FILE); exit 1; \
		fi; \
	fi

stop:
	@if $(RUNNING); then \
		pid=$$(cat $(PID_FILE)); kill $$pid; \
		while kill -0 $$pid 2>/dev/null; do sleep 0.1; done; \
		rm -f $(PID_FILE); echo "auth-api to'xtatildi"; \
	else \
		rm -f $(PID_FILE); echo "auth-api ishlamayapti"; \
	fi

# Graceful restart: joriy so'rovlar tugaydi, server yangi env bilan ko'tariladi
reload:
	@$(MAKE) --no-print-directory stop start

status:
	@if $(RUNNING); then \
		echo "auth-api ishlayapti (PID $$(cat $(PID_FILE)))"; \
	else \
		echo "auth-api ishlamayapti"; \
	fi

# Tashqi servicelar: ro'yxati $(COMPOSE_FILE) da
services-up:
	docker compose -f $(COMPOSE_FILE) up -d --wait

services-down:
	docker compose -f $(COMPOSE_FILE) down

services-status:
	docker compose -f $(COMPOSE_FILE) ps

# Testlar .env.local ga bog'liq emas: undagi o'zgaruvchilar test process'iga berilmaydi.
# INTEGRATION=1 bo'lsa integration testlar ham ishlaydi (testcontainers, Docker kerak).
test:
	@env $(addprefix -u ,$(ENV_VARS)) go test $(if $(INTEGRATION),-tags=integration) ./...
