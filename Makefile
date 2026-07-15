.PHONY: test bench demo

BINARY_NAME=logsan.exe
DEMO_INPUT=.//testdata/control/input
DEMO_OUTPUT=./testdata/control/output
DEMO_CONFIG=./testdata/control/config/detectors.yaml
DEMO_REPORT=./testdata/control/report.json

build:
	go build -o $(BINARY_NAME) ./cmd/logsan
# Тесты
test:
	go test -v ./...

# Бенчмарки
bench:
	go test -v ./... -bench=. -benchtime=1x

# Демонстрация
demo: build
	./$(BINARY_NAME) sanitize --in $(DEMO_INPUT) --out $(DEMO_OUTPUT) --config $(DEMO_CONFIG) --report $(DEMO_REPORT)
	

