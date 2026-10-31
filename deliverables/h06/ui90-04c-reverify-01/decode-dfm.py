import re, pathlib, json
raw = pathlib.Path(r'E:/19/herp/UI2.0/Logins/UserLogFrm.dfm').read_bytes()
text2 = raw.decode('latin1', errors='ignore')

def decode_hashes(s):
    return re.sub(r'#(\d+)', lambda m: chr(int(m.group(1))), s)

caps = re.findall(r'Caption = ([^\r\n]+)', text2)
print('CAPTIONS:')
for c in caps:
    print(decode_hashes(c))
print('---COLUMN_OBJECTS---')
for m in re.finditer(r'object (tvLogsColumn\d+):[\s\S]*?Caption = ([^\r\n]+)', text2):
    print(m.group(1), decode_hashes(m.group(2)))
