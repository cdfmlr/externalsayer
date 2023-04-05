#!python
# convert a xml file to a single line of text

import sys

if len(sys.argv) < 2:
    print('Usage: python xmlToSingleLine.py <xml file>')
    sys.exit(1)

with open(sys.argv[1]) as f:
    content = f.readlines()

content = [x.strip() for x in content]
content = [x for x in content if x != '']

if len(content) > 1:
    for i in range(len(content) - 1):
        if content[i][-1] != '>' or content[i + 1][0] != '<':
            content[i] += ' '

# print(' '.join(content))
print(''.join(content))
