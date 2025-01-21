
#'a' -> 'a' -> 'a' 
import json
import sys
import collections
import pathlib

if len(sys.argv) < 4:
    print("args: alphabetFile datasetFile outputFile")
    exit(1)

alphabetFile = sys.argv[1]
datasetFile = sys.argv[2]
outputFile = sys.argv[3]

chain = {}

alphabet =  pathlib.Path(alphabetFile).read_text()
for i1 in alphabet:
    chain[i1] = {}
    for i2 in alphabet:
        chain[i1][i2] = {}
        for i3 in alphabet:
            chain[i1][i2][i3] = 0
            

counter = 0

with pathlib.Path(datasetFile).open() as b:
    q = collections.deque()
    def readOne():
        while True:
            letter = b.read(1)
            if letter == '':
                return
            letter = letter.lower()
            if not letter in alphabet:
                continue
            return letter

    q.append(readOne())
    q.append(readOne())
    
    while True:
        l = readOne()
        if l is None: 
            break
        
        q.append(l)
        counter += 1
        chain[q[0]][q[1]][q[2]] += 1
        q.popleft()

for i1 in alphabet:
    for i2 in alphabet:
        for i3 in alphabet:
            chain[i1][i2][i3] = float(chain[i1][i2][i3]) / counter

result = {
    "alphabet": alphabet,
    "chain": chain
}
pathlib.Path(outputFile).write_text(json.dumps(result))
