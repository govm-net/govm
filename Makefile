
TIME:=$(shell date '+%Y%m%d-%H%M%S')
SOFTWARE:=govm-$(TIME)
OUTPUT:=output/$(SOFTWARE)

.PHONY: clean all 

all: 
	mkdir -p $(OUTPUT)
	# CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o $(OUTPUT)/govm.exe
	# CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(OUTPUT)/govm_amd.bin
	CGO_ENABLED=0 go build -o $(OUTPUT)/govm.exe
	cp run.tmpl $(OUTPUT)
	cp main.tmpl $(OUTPUT)
	mkdir -p $(OUTPUT)/conf
	cp conf/conf.json $(OUTPUT)/conf
	cp conf/bootstrap.json $(OUTPUT)/conf
	cp conf/event_filter.json $(OUTPUT)/conf
	cp -rf static $(OUTPUT)
	cp LICENSE $(OUTPUT)
	cd output && tar zcvf $(SOFTWARE).tar.gz $(SOFTWARE)

clean:
	go clean
