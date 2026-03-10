.PHONY: build build-windows clean

# Сборка для текущей ОС
build:
	go build -o garbage-collector ./cmd/garbage-collector

# Сборка для Windows (кросс-компиляция)
build-windows:
	CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc GOOS=windows GOARCH=amd64 go build -ldflags="-H windowsgui" -o garbage-collector.exe ./cmd/garbage-collector

# Очистка собранных файлов
clean:
	rm -f garbage-collector garbage-collector.exe

# По умолчанию сборка для текущей ОС
all: build