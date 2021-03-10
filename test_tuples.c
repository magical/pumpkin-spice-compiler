// +build ignore

// cc -o test_tuples test_tuples.c runtime.c -Wall -Wextra

#include <stddef.h>
#include <stdint.h>
#include <stdio.h>
#include <assert.h>

#include "runtime.h"

const int SIZE = 4*1024;

struct tuple {
	uint8_t len; // number of elements, max 63
	uint8_t isptr[63]; // whether each element is a pointer
	struct tuple* forwarding; // if not null, the forwarding address of the tuple 
	uintptr_t elem[]; // followed by len x uint64 values
};


int psc_main(void) {
	psc_gcinit(SIZE, SIZE);
	void** stack = rootstack_begin;
	struct tuple *t = psc_newtuple(stack, 2, 0x02);
	printf("len = %d\n", t->len);
	printf("t = %p\n", t);
	assert(t->isptr[0] == 0);
	assert(t->isptr[1] == 1);
	t->elem[0] = (uintptr_t)0xabad1dea;
	t->elem[1] = (uintptr_t)t;

	size_t inuse = 0, hsize;
	psc_gcgetsize(&inuse, &hsize);
	printf("heap alloc = %zd, size = %zd\n", inuse, hsize);

	*(struct tuple**)stack++ = t;

	printf("collect\n");
	psc_gccollect(stack);
	psc_gcgetsize(&inuse, &hsize);
	t = stack[-1];
	printf("heap alloc = %zd, size = %zd\n", inuse, hsize);
	printf("t = %p\n", t);

	stack--;

	printf("collect\n");
	psc_gccollect(stack);
	psc_gcgetsize(&inuse, &hsize);
	printf("heap alloc = %zd, size = %zd\n", inuse, hsize);

	// allocate a bunch of garbage values
	int cur = 0;
	const int N = sizeof(struct tuple) + sizeof(uintptr_t);
	while (cur+N < SIZE)  {
		psc_newtuple(stack, 1, 0);
		cur += N;
	}
	psc_gcgetsize(&inuse, &hsize);
	// this should show that we are almost at the max alloc!
	printf("heap alloc = %zd, size = %zd\n", inuse, hsize);
	printf("allocating\n");
	psc_newtuple(stack, 1, 0);
	psc_gcgetsize(&inuse, &hsize);
	// ...and this should show that we collected everything (except the new tuple)
	printf("heap alloc = %zd, size = %zd\n", inuse, hsize);

	//psc_gccollect(stack);
	printf("---\n");

	// allocate a linked list
	struct tuple* tail = NULL;
	*stack++ = (void*)tail;
	for (cur = inuse; cur+N < SIZE; cur += N)  {
		struct tuple* head = psc_newtuple(stack, 1, 0x1);
		head->elem[0] = (uintptr_t)tail;
		tail = head;
		stack[-1] = (void*)tail;
	}
	psc_gcgetsize(&inuse, &hsize);
	// this should show that we are almost at the max alloc!
	printf("heap alloc = %zd, size = %zd\n", inuse, hsize);
	// we have one garbage tuple on the heap
	// and the heap is full
	// so if we alloc one more, we should do a collection and
	// have enough space left for the allocation
	tail = psc_newtuple(stack, 1, 1);
	tail->elem[0] = (uintptr_t)stack[-1];
	stack[-1] = tail;
	psc_gcgetsize(&inuse, &hsize);
	printf("heap alloc = %zd, size = %zd\n", inuse, hsize);
	// now the heap is really full, so another allocation should
	// trigger a growth
	psc_newtuple(stack, 1, 0);
	psc_gcgetsize(&inuse, &hsize);
	printf("heap alloc = %zd, size = %zd\n", inuse, hsize);
	// let's allocate a few more and then do a full collection
	printf("alloc 3\n");
	for (int i = 0; i < 3; i++) {
		tail = psc_newtuple(stack, 1, 0);
		tail->elem[0] = (uintptr_t)stack[-1];
		stack[-1] = tail;
		psc_gccollect(stack);
	}
	stack--;
	psc_gccollect(stack);
	psc_gcgetsize(&inuse, &hsize);
	printf("heap alloc = %zd, size = %zd\n", inuse, hsize);

	return 0;
}
