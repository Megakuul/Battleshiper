.PHONY: build-BattleshiperApiWebFunc

build-BattleshiperApiWebFunc:
	bun install
	bun run build
	cp build/server/index.js "$(ARTIFACTS_DIR)/index.mjs"