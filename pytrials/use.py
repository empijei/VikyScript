from ctypes import cdll
lib = cdll.LoadLibrary('./libfoo.so')
print("Loaded go generated SO library")
result = lib.add(2, 3)
print(result)
result2 = lib.list()
print(result2)
