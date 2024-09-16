.PHONY: build-BattleshiperApiWebFunc

build-BattleshiperApiWebFunc:
	npm ci
	npm run build
	cp build/server/index.js "$(ARTIFACTS_DIR)/index.js"