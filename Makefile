CFLAGS ?= -Wall -Wextra -O2 -fcf-protection=none

test_tuples: runtime.c
test_tuples.c: runtime.h Makefile

runtime.h : runtime.c Makefile
	echo "// +build ignore" >$@
	sed -n -e '/^EXPORT /s//extern /p' <$< >>$@
