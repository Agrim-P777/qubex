#!/usr/bin/env python3
"""Assemble actual license texts for modules linked into qbx."""
import json
import pathlib
import subprocess
text=subprocess.check_output(['go','list','-deps','-json','./cmd/qbx'],text=True)
decoder=json.JSONDecoder()
modules={}
while text.strip():
    text=text.lstrip()
    obj,end=decoder.raw_decode(text)
    text=text[end:]
    module=obj.get('Module',{})
    if module and not module.get('Main'):
        modules[module['Path']]=module
print('Qubex third-party dependency notices\n')
for name,module in sorted(modules.items()):
    root=pathlib.Path(module['Dir'])
    candidates=sorted({*root.glob('LICENSE*'),*root.glob('COPYING*')})
    files=[p for p in candidates if p.is_file()]
    if not files:
        raise SystemExit(f'Missing license text for {name}; inspect before distributing')
    print('='*72, f'\n{name} {module["Version"]}\n')
    for p in files:
        print(p.read_text())
