.PHONY: run build test clean db-check mail-test verify-check all-checks

run:
	go run ./cmd/server

build:
	go build -o bin/server ./cmd/server

test:
	go test ./...

ifeq ($(OS),Windows_NT)
clean:
	@if exist bin rmdir /s /q bin
else
clean:
	@rm -rf bin/
endif

# ─── Individual checks ──────────────────────────────────────────────────
# Each check requires specific env vars (see .env.example). If the required
# vars are missing the tool exits with a clear message.

db-check:
	go run ./cmd/tests/dbcheck

mail-test:
	go run ./cmd/tests/mailtest

verify-check:
	go run ./cmd/tests/verifycheck

# ─── Run all configured checks ──────────────────────────────────────────
#   CHECKS=db              → only database
#   CHECKS=db,verify       → database + email verification
#   CHECKS=db,mail,verify  → all three (default)

ifeq ($(OS),Windows_NT)
# Windows: run all three unconditionally (batch scripting for CHECKS
# filtering is too fragile). Use individual targets for specificity.
all-checks:
	$(MAKE) db-check
	$(MAKE) verify-check
	$(MAKE) mail-test
else
all-checks:
	@if [ -z "$$CHECKS" ]; then \
		$(MAKE) db-check && $(MAKE) verify-check && $(MAKE) mail-test || exit 1; \
	else \
		passed=0; failed=0; \
		for check in $$(echo $$CHECKS | tr ',' ' '); do \
			case "$$check" in \
				db)     echo "--- db-check ---";     $(MAKE) db-check      && passed=$$((passed+1)) || failed=$$((failed+1)) ;; \
				mail)   echo "--- mail-test ---";    $(MAKE) mail-test     && passed=$$((passed+1)) || failed=$$((failed+1)) ;; \
				verify) echo "--- verify-check ---"; $(MAKE) verify-check  && passed=$$((passed+1)) || failed=$$((failed+1)) ;; \
				*) echo "unknown check: $$check (use db, mail, verify)";; \
			esac; \
		done; \
		echo ""; \
		echo "checks: $$passed passed, $$failed failed"; \
		[ $$failed -eq 0 ] || exit 1; \
	fi
endif
