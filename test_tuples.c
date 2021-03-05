// +build ignore

// cc -o test_tuples test_tuples.c runtime.c -Wall -Wextra

#include <stddef.h>
#include <stdint.h>
#include <stdio.h>

#include "runtime.h"

const int SIZE = 16*1024;

struct tuple {
	uint8_t len; // number of elements, max 63
	uint8_t isptr[63]; // whether each element is a pointer
	struct tuple* forwarding; // if not null, the forwarding address of the tuple 
	uintptr_t elem[]; // followed by len x uint64 values
};


int main() {
	psc_gcinit(SIZE, SIZE);
	struct tuple *t = psc_newtuple(1);
	printf("%d\n", t->len);
	return 0;
}
