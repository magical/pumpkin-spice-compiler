// +build ignore

#include <stdint.h>
#include <stdlib.h>
#include <stdio.h>
#include <assert.h>

int psc_main(void);

#ifndef NOMAIN
int main(int argc, char**argv) {
	int result = psc_main();
	printf("%d\n", result);
	return 0;
}
#endif


/* cheney 2-space copying collector */

#define EXPORT
EXPORT void psc_gcinit(size_t stack_size, size_t heap_size);
EXPORT void psc_gccollect(void** rootstack_ptr, size_t bytes_requested);
EXPORT void psc_gcgetsize(size_t* heap_inuse_size, size_t* heap_size);
EXPORT struct tuple* psc_newtuple(int nelem);
void *free_ptr;
void *fromspace_begin;
void *fromspace_end;
void *tospace_begin;
void *tospace_end;
EXPORT void **rootstack_begin;

// Initializes the garbarge collector.
// Allocates stack_size bytes for the pointer stack (shadow stack)
// and heap_size bytes for the heap.
void psc_gcinit(size_t stack_size, size_t heap_size)
{
	stack_size += -stack_size&63;
	heap_size += -heap_size&63;
	rootstack_begin = calloc(stack_size, 1);
	fromspace_begin = calloc(heap_size, 1);
	fromspace_end = (char*)fromspace_begin + heap_size;
	tospace_begin = calloc(heap_size, 1);
	tospace_end = (char*)tospace_begin + heap_size;
	free_ptr = fromspace_begin;
}

void psc_gcgetsize(size_t *heap_inuse_size, size_t *heap_size)
{
	if(heap_size) *heap_size = (size_t)(fromspace_end - fromspace_begin);
	if(heap_inuse_size) *heap_inuse_size = (size_t)(free_ptr - fromspace_begin);
}

struct tuple {
	uint8_t len; // number of elements, max 63
	uint8_t isptr[63]; // whether each element is a pointer
	struct tuple* forwarding; // if not null, the forwarding address of the tuple
	uintptr_t elem[]; // followed by len x uint64 values
};

// Collects unreachable objects and ensures enough free space for an allocation
// of bytes_requested.
//
// Copies the heap from fromspace to tospace
void psc_gccollect(void** rootstack_ptr, size_t bytes_requested)
{
	// TODO: reallocate tospace?

	// these two pointers will track our progress
	// scan_ptr points to the beginning of our queue of items to be scanned/copied
	// and end_ptr points to the end of the queue and the beginning of the free space
	void *scan_ptr, *end_ptr;
	scan_ptr = tospace_begin;
	end_ptr = tospace_begin;

	// first step:
	// iterate over the root stack
	// copy each tuple to tospace
	for (void **p = rootstack_begin; p < rootstack_ptr; p++) {
		// copy tuple to tospace
		// this is a shallow copy - we don't recursively copy
		// any other tuples yet, nor do we update any pointers
		struct tuple* oldptr = *p;
		struct tuple* newptr = end_ptr;
		*newptr = *oldptr;
		assert(newptr->len <= 63);
		assert(newptr->forwarding == NULL);
		for (int i = 0; i < oldptr->len; i++) {
			newptr->elem[i] = oldptr->elem[i];
		}
		end_ptr = (struct tuple*)end_ptr + 1;
		end_ptr = (uintptr_t*)end_ptr + oldptr->len;
	}

	// graph copy:
	// use tospace as both our queue of to-be-copied items
	// and as our destination for copied items.
	//
	// oh, interesting. this algorithm assumes an absence of interior pointers
	// (any references to a tuple must point to the beginning of that tuple,
	// not to an element within it)
	while (scan_ptr < end_ptr && scan_ptr < tospace_end) {
		struct tuple* cur = scan_ptr;
		// walk over the current tuple looking for pointers
		// they should all point to the old space
		// if the pointed-at tuple has a forwarding address, update this pointer
		// otherwise copy the old object to the end of the queue
		assert(cur->len <= 63);
		for (int i = 0; i < cur->len; i++) {
			if (!cur->isptr[i]) {
				continue;
			}
			struct tuple* oldptr = (struct tuple*)cur->elem[i];
			assert(!(tospace_begin <= (void*)oldptr && (void*)oldptr < tospace_end));
			assert(fromspace_begin <= (void*)oldptr && (void*)oldptr < fromspace_end);
			if (oldptr->forwarding != NULL) {
				cur->elem[i] = (uintptr_t)oldptr->forwarding;
				continue;
			}
			// copy tuple to tospace (shallow copy)
			struct tuple* newptr = end_ptr;
			*newptr = *oldptr;
			assert(newptr->len <= 63);
			assert(newptr->forwarding == NULL);
			for (int i = 0; i < oldptr->len; i++) {
				newptr->elem[i] = oldptr->elem[i];
			}
			// advance end_ptr
			end_ptr = (struct tuple*)end_ptr + 1;
			end_ptr = (uintptr_t*)end_ptr + oldptr->len;
			// set forwarding address
			oldptr->forwarding = newptr;
		}
		// advance scan_ptr
		scan_ptr = (struct tuple*)scan_ptr + 1;
		scan_ptr = (uintptr_t*)scan_ptr + cur->len;
	}

	// swap tospace and fromspace
	void* tmp = fromspace_begin;
	fromspace_begin = tospace_begin;
	tospace_begin = tmp;

	tmp = fromspace_end;
	fromspace_end = tospace_end;
	tospace_end = tmp;

	free_ptr = end_ptr;

	// TODO: update root pointers

	// if there isn't enough space
	// allocate a new space and do another copy
	// realloc if there isn't enough
	size_t free_space = free_ptr - fromspace_begin;
	if (free_space < bytes_requested) {
		//...

		// technically we don't need to do a full collection -
		// we could do a memcpy followed by a pass to fix the pointers
		// but that requires more code and this is easier
	}
}

void* psc_alloc(size_t size)
{
	if ((size_t)((char*)fromspace_end - (char*)free_ptr) < size) {
		psc_gccollect(rootstack_begin, size);
	}
	void* mem = free_ptr;
	free_ptr = (char*)free_ptr + size;
	return mem;
}

struct tuple* psc_newtuple(int nelem)
{
	size_t size = sizeof(struct tuple) + nelem*sizeof(uintptr_t);
	if ((size_t)((char*)fromspace_end - (char*)free_ptr) < size) {
		psc_gccollect(rootstack_begin, size);
	}
	struct tuple* new = free_ptr;
	new->len = nelem;
	new->forwarding = NULL;
	free_ptr = (char*)free_ptr + size;
	return new;
}
