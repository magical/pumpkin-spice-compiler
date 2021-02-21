// +build ignore

#include <stdio.h>

int psc_main(void);

int main(int argc, char**argv) {
	int result = psc_main();
	printf("%d\n", result);
	return 0;
}
