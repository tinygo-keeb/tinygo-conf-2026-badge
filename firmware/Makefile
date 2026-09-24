TARGET ?= esp32s3-box-3
EXAMPLES := $(notdir $(wildcard examples/*))

.PHONY: all build $(EXAMPLES) flash-%

all: build

# 全 example をビルドして生成物は捨てる (コンパイル確認用)
build:
	@for e in $(EXAMPLES); do \
		echo "== $$e"; \
		tinygo build --target $(TARGET) --size short -o /dev/null ./examples/$$e || exit 1; \
	done

# 例: make flash-blink
flash-%:
	tinygo flash --target $(TARGET) --size short ./examples/$*

# 例: make monitor
monitor:
	tinygo monitor --target $(TARGET)

