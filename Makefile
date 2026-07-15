.PHONY: test bench demo

BINARY_NAME=logsan.exe
DEMO_INPUT=./testdata/control/input/
DEMO_OUTPUT=./testdata/control/output/
DEMO_CONFIG=./testdata/control/config/detectors.yaml
DEMO_REPORT=./testdata/control/report.json

build:
	go build -o $(BINARY_NAME) ./cmd/logsan
# Тесты
test:
	go test -v ./...

# Бенчмарки
bench:
	go test -v ./... -bench=BenchmarkSmall -benchmem

#Бенчмарк на 1 ГБ лога
bench-1gb:
	LOGSAN_BENCH_1GB=1 go test ./... -bench=Benchmark_GBLog -benchtime=1x

# Демонстрация
demo: build
	./$(BINARY_NAME) sanitize --in $(DEMO_INPUT) --out $(DEMO_OUTPUT) --config $(DEMO_CONFIG) --report $(DEMO_REPORT)
	

