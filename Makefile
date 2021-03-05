CFLAGS ?= -Wall -Wextra

test_tuples : CFLAGS += -DNOMAIN
test_tuples: runtime.c
test_tuples.c: runtime.h

runtime.h : runtime.c Makefile
	echo "// +build ignore" >$@
	sed -n -e '/^EXPORT /s//extern /p' <$< >>$@
