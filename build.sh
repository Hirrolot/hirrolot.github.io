#!/bin/bash

gcc gen.c -o gen -Wall -Wextra -pedantic -std=gnu99
./gen
make all
