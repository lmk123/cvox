.PHONY: install build test clean

# 安装到 ~/go/bin（本地开发用，使用当前工作目录代码）
install:
	go install .

# 安装最新发布版本（对应 @latest）
install-latest:
	go install .@latest

# 编译到 bin/cvox-bin（与 npm 安装路径一致）
build:
	go build -o bin/cvox-bin .

# 运行测试
test:
	go test ./...

# 清理
clean:
	rm -f bin/cvox-bin
