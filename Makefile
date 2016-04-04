execs = cli discosrv gridsdr resman

.PHONY: clean $(execs)

all: $(execs)

$(execs):
	mkdir -p bin
	go build -v  -o bin/$@ github.com/kc1212/virtual-grid/cmd/$@

clean:
	rm -rf bin
