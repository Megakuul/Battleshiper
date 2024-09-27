.PHONY: build-BattleshiperApiWebFunc

build-BattleshiperApiWebFunc:
	bun install
	bun run build
	mkdir -p "$(ARTIFACTS_DIR)/prerendered" "$(ARTIFACTS_DIR)/client"
	cp -r build/prerendered/* "$(ARTIFACTS_DIR)/prerendered"
	cp -r build/client/* "$(ARTIFACTS_DIR)/client"
	cp build/server/index.js "$(ARTIFACTS_DIR)/index.mjs"