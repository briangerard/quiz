EXE=${GOPATH}/bin/compound
SRC=compound.go

all: $(EXE)

$(EXE): $(SRC)
	go build -o $(EXE) -i $(SRC)


test:
	go test -v
