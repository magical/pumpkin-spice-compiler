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
	void** stack = rootstack_begin;
	struct tuple *t = psc_newtuple(2);
	printf("len = %d\n", t->len);
	printf("t = %p\n", t);
	t->isptr[0] = 0;
	t->elem[0] = (uintptr_t)0xabad1dea;
	t->elem[1] = (uintptr_t)t;

	*(struct tuple**)stack++ = t;

	printf("collect\n");
	psc_gccollect(stack, 0);
	size_t inuse;
	psc_gcgetsize(&inuse, NULL);
	printf("heap alloc = %zd\n", inuse);
	printf("t = %p\n", t); //uhh wait

	stack--;

	printf("collect\n");
	psc_gccollect(stack, 0);
	psc_gcgetsize(&inuse, NULL);
	printf("heap alloc = %zd\n", inuse);
	return 0;
}
