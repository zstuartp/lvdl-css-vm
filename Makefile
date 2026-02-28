.PHONY: examples examples-tiny size-report size-check prepublish

SPEC ?= examples/standard.lvdl
TINY_SPEC ?= examples/standard-tiny.lvdl
LVDLC ?= go run ./cmd/lvdlc
PROFILE ?= runtime8

SIZE_BUDGET_ADD_CSS ?= 300000
SIZE_BUDGET_ADD_JS ?= 305000
SIZE_BUDGET_JNZ_CSS ?= 430000
SIZE_BUDGET_JNZ_JS ?= 435000
SIZE_BUDGET_SMOKE_CSS ?= 920000
SIZE_BUDGET_SMOKE_JS ?= 925000

EXAMPLE_BASES := \
	examples/add_demo \
	examples/overflow_demo \
	examples/sub_demo \
	examples/and_demo \
	examples/xor_demo \
	examples/or_demo \
	examples/dup_demo \
	examples/swp_demo \
	examples/ovr_demo \
	examples/inc_demo \
	examples/dec_demo \
	examples/cll_ret_demo \
	examples/pop_demo \
	examples/not_demo \
	examples/shl_demo \
	examples/shr_demo \
	examples/jmp_demo \
	examples/jnz_demo \
	examples/isa_smoke

TINY_EXAMPLE_BASES := \
	examples/tiny_add_demo \
	examples/tiny_jnz_demo

examples:
	@for b in $(EXAMPLE_BASES); do \
		$(LVDLC) -spec $(SPEC) -asm $${b}.asm -profile $(PROFILE) -mode live-pure-css -out $${b}-css-pure.html; \
		$(LVDLC) -spec $(SPEC) -asm $${b}.asm -profile $(PROFILE) -mode live-js-clock -out $${b}-js-clock.html; \
	done

examples-tiny:
	@for b in $(TINY_EXAMPLE_BASES); do \
		$(LVDLC) -spec $(TINY_SPEC) -asm $${b}.asm -profile runtime8-tiny -mode live-pure-css -out $${b}-css-pure.html; \
		$(LVDLC) -spec $(TINY_SPEC) -asm $${b}.asm -profile runtime8-tiny -mode live-js-clock -out $${b}-js-clock.html; \
	done

size-report:
	@tmp=$$(mktemp -d); \
	trap 'rm -rf "$$tmp"' EXIT; \
	printf "%-14s %11s %11s %11s %8s\n" "example" "ctrl(bytes)" "none(bytes)" "saved" "saved%"; \
	printf "%-14s %11s %11s %11s %8s\n" "--------------" "-----------" "-----------" "-----------" "-------"; \
	total_ctrl=0; \
	total_none=0; \
	for b in $(EXAMPLE_BASES); do \
		base=$$(basename $$b); \
		ctrl="$$tmp/$${base}-ctrl.html"; \
		none="$$tmp/$${base}-none.html"; \
		$(LVDLC) -spec $(SPEC) -asm $${b}.asm -profile $(PROFILE) -mode live-pure-css -opt control-state -out $$ctrl >/dev/null; \
		$(LVDLC) -spec $(SPEC) -asm $${b}.asm -profile $(PROFILE) -mode live-pure-css -opt none -out $$none >/dev/null; \
		ctrl_bytes=$$(wc -c < $$ctrl | tr -d ' '); \
		none_bytes=$$(wc -c < $$none | tr -d ' '); \
		saved=$$((none_bytes-ctrl_bytes)); \
		saved_pct=$$(awk -v c=$$ctrl_bytes -v n=$$none_bytes 'BEGIN{if(n==0){printf "0.0"}else{printf "%.1f",((n-c)*100.0)/n}}'); \
		printf "%-14s %11d %11d %11d %7s%%\n" "$$base" "$$ctrl_bytes" "$$none_bytes" "$$saved" "$$saved_pct"; \
		total_ctrl=$$((total_ctrl+ctrl_bytes)); \
		total_none=$$((total_none+none_bytes)); \
	done; \
	total_saved=$$((total_none-total_ctrl)); \
	total_saved_pct=$$(awk -v c=$$total_ctrl -v n=$$total_none 'BEGIN{if(n==0){printf "0.0"}else{printf "%.1f",((n-c)*100.0)/n}}'); \
	printf "%-14s %11s %11s %11s %8s\n" "--------------" "-----------" "-----------" "-----------" "-------"; \
	printf "%-14s %11d %11d %11d %7s%%\n" "total" "$$total_ctrl" "$$total_none" "$$total_saved" "$$total_saved_pct"

size-check:
	@set -e; \
	fail=0; \
	check(){ \
		file="$$1"; \
		budget="$$2"; \
		actual=$$(wc -c < "$$file" | tr -d ' '); \
		if [ "$$actual" -gt "$$budget" ]; then \
			echo "size-check: FAIL $$file actual=$$actual budget=$$budget"; \
			fail=1; \
		else \
			echo "size-check: ok   $$file actual=$$actual budget=$$budget"; \
		fi; \
	}; \
	check examples/add_demo-css-pure.html $(SIZE_BUDGET_ADD_CSS); \
	check examples/add_demo-js-clock.html $(SIZE_BUDGET_ADD_JS); \
	check examples/jnz_demo-css-pure.html $(SIZE_BUDGET_JNZ_CSS); \
	check examples/jnz_demo-js-clock.html $(SIZE_BUDGET_JNZ_JS); \
	check examples/isa_smoke-css-pure.html $(SIZE_BUDGET_SMOKE_CSS); \
	check examples/isa_smoke-js-clock.html $(SIZE_BUDGET_SMOKE_JS); \
	if [ "$$fail" -ne 0 ]; then \
		exit 1; \
	fi

prepublish:
	@echo "==> go test"
	@go test ./...
	@echo "==> make examples"
	@$(MAKE) examples
	@echo "==> make examples-tiny"
	@$(MAKE) examples-tiny
	@echo "==> make size-check"
	@$(MAKE) size-check
	@echo "==> make size-report"
	@$(MAKE) size-report
