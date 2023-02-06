MAKEFLAGS  := --silent --always-make
PAR        := $(MAKE) -j 128
VERB       := $(if $(filter $(verb), true), -v,)
FAIL       := $(if $(filter $(fail), false),, -failfast)
SHORT      := $(if $(filter $(short), true), -short,)
TEST_FLAGS := -count=1 $(VERB) $(FAIL) $(SHORT)
TEST       := test $(TEST_FLAGS) -timeout=1s -run=$(run)
BENCH      := test $(TEST_FLAGS) -run=- -bench=$(or $(run),.) -benchmem -benchtime=128ms
WATCH      := watchexec -r -c -d=0 -n

default: test_w

watch:
	$(PAR) test_w lint_w

test_w:
	gow -c -v $(TEST)

test:
	go $(TEST)

bench_w:
	gow -c -v $(BENCH)

bench:
	go $(BENCH)

lint_w:
	$(WATCH) -- $(MAKE) lint

lint:
	golangci-lint run
	echo [lint] ok
