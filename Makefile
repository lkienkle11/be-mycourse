.PHONY: build build-nocgo

# Production build: requires CGO_ENABLED=1 and libvips-dev.
# On Ubuntu VPS: sudo apt-get install -y libvips-dev pkg-config
build:
#	go run ./cmd/syncpermissions # sẽ chạy trong tương lai
	CGO_ENABLED=1 go build -trimpath -buildvcs=false -o /dev/null .

# Pure-Go build for environments without libvips (local code check, review).
# Image uploads at runtime will return errcode 9017 (ImageEncodeBusy); all other features unaffected.
build-nocgo:
	CGO_ENABLED=0 go build -trimpath -buildvcs=false -o /dev/null .

.PHONY: check-architecture

check-architecture:
	@echo "🔍 Scanning 'services/' and 'repository/' directory structure..."
	@for target_dir in services repository; do \
		if [ ! -d "$$target_dir" ]; then \
			continue; \
		fi; \
		ROOT_FILES=$$(find $$target_dir/ -maxdepth 1 -name "*.go" 2>/dev/null | wc -l | tr -d ' '); \
		if [ "$$ROOT_FILES" -gt 3 ]; then \
			echo "🚨 ARCHITECTURE ERROR: Root directory '$$target_dir/' contains $$ROOT_FILES .go files!"; \
			echo "👉 Maximum limit of 3 files exceeded. Please group modules into business subfolders."; \
			exit 1; \
		fi; \
		SUBFOLDERS=$$(find $$target_dir/ -mindepth 1 -maxdepth 1 -type d 2>/dev/null); \
		if [ -z "$$SUBFOLDERS" ] && [ "$$ROOT_FILES" -eq 0 ]; then \
			echo "⚠️ The '$$target_dir/' directory is currently empty."; \
			continue; \
		fi; \
		for dir in $$SUBFOLDERS; do \
			FOLDER_NAME=$$(basename $$dir); \
			if ! echo "$$FOLDER_NAME" | grep -Eq '^[a-z0-9_]+$$'; then \
				echo "🚨 NAMING ERROR: Subfolder '$$FOLDER_NAME' in '$$target_dir/' is invalid!"; \
				echo "👉 Business subfolder names must be lowercase, without spaces, using underscores (e.g., user_auth, order_mgmt)."; \
				exit 1; \
			fi; \
			FILE_IN_SUB=$$(find "$$dir" -name "*.go" | wc -l | tr -d ' '); \
			if [ "$$FILE_IN_SUB" -eq 0 ]; then \
				echo "🚨 MODULE ERROR: Business subfolder '$$dir' is empty!"; \
				echo "👉 Created subfolders must contain corresponding processing modules."; \
				exit 1; \
			fi; \
		done; \
	done
	@echo "✅ Architecture is valid for both 'services/' and 'repositories/' (Max 3 root files, standard business subfolders)!"