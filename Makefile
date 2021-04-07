all:
	CGO_ENABLED=0 go build -o alias-resolve ./
	upx -9 alias-resolve
	upx -t alias-resolve
