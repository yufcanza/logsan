# Makefile для logsan
.PHONY: test bench demo

BINARY_NAME=logsan
DEMO_INPUT=./testdata/in
DEMO_OUTPUT=./testdata/out
DEMO_CONFIG=detectors.yaml
DEMO_REPORT=./testdata/report.json

build:
	go build .
# Тесты
test:
	go test -v ./...

# Бенчмарки
bench:
	go test -v ./... -bench=. -benchtime=1x

# Демонстрация
demo: build
	./$(BINARY_NAME) sanitize --in $(DEMO_INPUT) --out $(DEMO_OUTPUT) --config $(DEMO_CONFIG) --report $(DEMO_REPORT)
	

