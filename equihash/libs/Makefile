CPP=g++
CFLAGS=-std=c++11 -O2

TARGETS=libequi.a

.PHONY: clean

all: libequi.a

equi200_9.o: equi.c
	$(CPP) -D WN=200 -D WK=9 -c $(CFLAGS) $^ -o $@

equi48_5.o: equi.c
	$(CPP) -D WN=48 -D WK=5 -c $(CFLAGS) $^ -o $@

blake2b.o: blake/blake2b.cpp
	$(CPP) -c $(CFLAGS) $^

libequi.a: equi200_9.o equi48_5.o blake2b.o
	ar rcs $@ $^

clean:
	rm -f equi.o blake2b.o libequi.a
