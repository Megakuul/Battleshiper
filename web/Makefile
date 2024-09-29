.PHONY: build-BattleshiperApiWebFunc

# This build process must write asset artifacts directly to the bootstrap function.
# Reason: Each sveltekit build generates a different set of asset and server outputs that must match (read more in /web/hooks/README.md).
BOOTSTRAP_ARTIFACTS_DIR = $(ARTIFACTS_DIR)/../BattleshiperApiWebBootstrapFunc

build-BattleshiperApiWebFunc:
	bun install
	bun run build
	mkdir -p "$(BOOTSTRAP_ARTIFACTS_DIR)/prerendered" "$(BOOTSTRAP_ARTIFACTS_DIR)/client"
	cp -r build/prerendered/* "$(BOOTSTRAP_ARTIFACTS_DIR)/prerendered"
	cp -r build/client/* "$(BOOTSTRAP_ARTIFACTS_DIR)/client"
	cp build/server/index.js "$(ARTIFACTS_DIR)/index.mjs"